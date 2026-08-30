package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/disintegration/imaging"
)

// ===================== 圆角自动识别 =====================

// pixelAt 读取 (x, y) 处的像素色
func pixelAt(img image.Image, bounds image.Rectangle, x, y int) color.RGBA {
	r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// isBg 判断像素是否属于角部背景（透明或与背景色接近）
// 阈值 12：支持"不透明浅色圆角 + 略深主体"的低对比场景（如 iOS 风格浅蓝白图标）
func isBg(c color.RGBA, bg color.RGBA) bool {
	return c.A < 128 || colorDistance(c, bg) <= 12
}

// DetectCornerRadius 自动检测图片四角是否有多余背景（圆角外区域），
// 返回建议的切割圆角半径（原图像素）和是否检测到。
// 原理：
//  1. 角部背景色需与中心内容色差异明显；
//  2. 沿对角线扫描找到"背景 -> 主图"转折点 t（t = pad + r·0.293）；
//  3. 沿边缘逐行扫描，最大转折距离 e = pad + r；
//  4. 联立解出 r = (e - t) / 0.707，兼容图标四周有留白的情况。
func DetectCornerRadius(img image.Image) (radius int, detected bool) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 40 || h < 40 {
		return 0, false
	}

	// 中心内容色参考
	content := averageColor(img, w/2-4, h/2-4, 8, 8)

	// 四个角：{角点x, 角点y, 向内方向dx, 向内方向dy}
	corners := [4][4]int{
		{0, 0, 1, 1}, {w - 1, 0, -1, 1}, {0, h - 1, 1, -1}, {w - 1, h - 1, -1, -1},
	}

	radii := []int{}
	for _, c := range corners {
		cx, cy, dx, dy := c[0], c[1], c[2], c[3]

		// 角部背景色（角内 1~3px 采样）
		bg := averageColor(img, cx+dx, cy+dy, 3, 3)

		// 背景与中心内容差异必须明显，否则该角没有可切背景
		// 阈值 8：兼容浅色圆角（如白底→浅蓝主体，色差~12）
		if colorDistance(bg, content) < 8 {
			continue
		}

		// 对角线扫描：转折点 t
		t, ok := scanCornerDiagonal(img, bounds, cx, cy, dx, dy, bg)
		if !ok || t < 3 {
			continue
		}

		// 角部背景均匀性校验（排除照片/渐变背景误判）
		if !cornerBgUniform(img, bounds, cx, cy, dx, dy, bg, t) {
			continue
		}

		// 边缘扫描：最大转折 e = pad + r
		e, ok2 := scanCornerEdges(img, bounds, cx, cy, dx, dy, bg)
		if !ok2 || e <= t {
			continue
		}

		r := int(float64(e-t)/0.7071) + 1
			if r > 0 {
				// 沿圆角弧段 [t, e] 采样边界点做最小二乘圆拟合精化。
				// 合理范围：拟合半径不应超出 e（圆角弧止于该位置）。
				if rf, okf := fitCornerArc(img, bounds, cx, cy, dx, dy, bg, t, e); okf && rf >= 2 && rf <= e {
					r = rf
				}
				radii = append(radii, r)
			}
	}
	if len(radii) == 0 {
		return 0, false
	}

	// 取中位数 + 10% 余量（把抗锯齿边缘也切干净）
	sort.Ints(radii)
	median := radii[len(radii)/2]
	result := median + median/10
	maxR := int(math.Min(float64(w), float64(h))) / 2
	if result > maxR {
		result = maxR
	}
	// 有效半径下限：小于图像边长 3% 的圆角基本是噪点/渐变误判，直接拒绝
	minR := int(math.Min(float64(w), float64(h))) / 33 // ≈3%
	if result < minR {
		return 0, false
	}
	if result < 4 {
		result = 4
	}
	return result, true
}

// scanCornerDiagonal 从角点沿 45° 对角线向内扫描，返回转折点的轴向距离 t
func scanCornerDiagonal(img image.Image, bounds image.Rectangle, cx, cy, dx, dy int, bg color.RGBA) (int, bool) {
	w := bounds.Dx()
	h := bounds.Dy()
	// 扫描上限放大到 1000：支持大半径圆角（>200px），原 200 cap 在 1024 图 20% 半径时刚好不够
	limit := int(math.Min(float64(w), float64(h))) / 3
	if limit > 1000 {
		limit = 1000
	}
	for step := 1; step < limit; step++ {
		x := cx + dx*step
		y := cy + dy*step
		if x < 0 || x >= w || y < 0 || y >= h {
			return 0, false
		}
		if !isBg(pixelAt(img, bounds, x, y), bg) {
			// 抗噪：后续连续 2 点仍非背景才判定为转折，避免孤立噪点误判
			ok := true
			for k := 1; k <= 2; k++ {
				nx := cx + dx*(step+k)
				ny := cy + dy*(step+k)
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					break
				}
				if isBg(pixelAt(img, bounds, nx, ny), bg) {
					ok = false
					break
				}
			}
			if ok {
				// 突变性校验：t-1 像素必须仍"明显接近背景"（distance < 8）。
				// 真实圆角的过渡是突变的（透明→不透明），t-1 几乎与背景一致；
				// 渐变图像的过渡是平滑的，t-1 距背景已超阈值 → 视为渐变，跳过。
				if step > 1 {
					prev := pixelAt(img, bounds, cx+dx*(step-1), cy+dy*(step-1))
					if colorDistance(prev, bg) > 8 {
						continue
					}
				}
				return step, true
			}
			// 视为噪点，继续向内扫描
		}
	}
	return 0, false
}

// cornerBgUniform 校验角部背景区域颜色均匀（真实纯色背景），排除渐变/照片
func cornerBgUniform(img image.Image, bounds image.Rectangle, cx, cy, dx, dy int, bg color.RGBA, t int) bool {
	// 透明背景：跳过均匀性校验。角部本来就是全透的，曲线抗锯齿的半透像素会被阈值误判。
	if bg.A < 20 {
		return true
	}
	w := bounds.Dx()
	h := bounds.Dy()
	// 采样点都限制在背景区域内（对角线转折点 t 以内）
	pts := [][2]int{
		{t / 2, t / 2},
		{t * 9 / 10, 3},
		{3, t * 9 / 10},
		{t * 3 / 5, t * 3 / 5},
	}
	for _, p := range pts {
		x := cx + dx*p[0]
		y := cy + dy*p[1]
		if x < 0 || x >= w || y < 0 || y >= h {
			continue
		}
		if colorDistance(pixelAt(img, bounds, x, y), bg) > 15 {
			return false
		}
	}
	return true
}

// scanCornerEdges 沿角部两条边逐行/列扫描，返回最大转折距离 e（= pad + r）
func scanCornerEdges(img image.Image, bounds image.Rectangle, cx, cy, dx, dy int, bg color.RGBA) (int, bool) {
	w := bounds.Dx()
	h := bounds.Dy()
	// 扫描上限改为 h/2、w/2，cap 1000：原 /3 + 200 cap 在 1024 图 20% 半径时刚好够不到 col=200
	rowLimit := h / 2
	if rowLimit > 1000 {
		rowLimit = 1000
	}
	colLimit := w / 2
	if colLimit > 1000 {
		colLimit = 1000
	}
	best := 0
	found := false
	for row := 0; row < rowLimit; row++ {
		y := cy + dy*row
		if y < 0 || y >= h {
			break
		}
		for col := 0; col < colLimit; col++ {
			x := cx + dx*col
			if x < 0 || x >= w {
				break
			}
			if !isBg(pixelAt(img, bounds, x, y), bg) {
				if col > best {
					best = col
				}
				found = true
				break
			}
		}
	}
	return best, found
}

// fitCornerArc 沿圆角弧段采样色块（前景）外边界，做最小二乘圆弧拟合。
// 扫描策略：沿"两条边"方向从角点向内逐行/列找前景边界点。
// 范围使用 [t, maxE]：t 之前是角部纯背景；maxE 是保守上界（避免扫描太深导致包含内部色块）。
//
// 关键改进：扫描方向固定从"边"出发，遇到前景就停——不论该行/列是直线段还是弧段，
// 只要该行/列上"背景 → 前景"转折点都参与拟合，规避真实图中内部镂空/渐变的干扰。
// 用一般圆方程 (x-A)²+(y-B)²=R² 做最小二乘，兼容 pad 不等、背景非纯色等真实图片。
func fitCornerArc(img image.Image, bounds image.Rectangle, cx, cy, dx, dy int, bg color.RGBA, t, maxE int) (int, bool) {
	w := bounds.Dx()
	h := bounds.Dy()
	if t < 2 || maxE <= t {
		return 0, false
	}
	// 扫描范围：从 t 到 maxE（maxE 取较松的上界：e + r/2，避免裁剪真实圆角）
	maxU := maxE + (maxE-t)/2
	if maxU >= w {
		maxU = w - 1
	}
	if maxU >= h {
		maxU = h - 1
	}
	minU := t
	if minU >= maxU {
		return 0, false
	}

	var xs, ys []float64
	// 沿两条边方向各采样：水平行（y 取 minU..maxU）找 x 拐点；垂直列（x 取 minU..maxU）找 y 拐点
	for v := minU; v <= maxU; v++ {
		py := cy + dy*v
		if py < 0 || py >= h {
			break
		}
		if u, ok := findEdgeAlongRow(img, bounds, cx, cy, dx, dy, v, bg); ok {
			xs = append(xs, float64(u))
			ys = append(ys, float64(v))
		}
	}
	for u := minU; u <= maxU; u++ {
		px := cx + dx*u
		if px < 0 || px >= w {
			break
		}
		if v, ok := findEdgeAlongCol(img, bounds, cx, cy, dx, dy, u, bg); ok {
			xs = append(xs, float64(u))
			ys = append(ys, float64(v))
		}
	}
	if len(xs) < 5 {
		if os.Getenv("ICONFORGE_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "fitCornerArc: 样本不足 (n=%d) 角=(%d,%d) dir=(%d,%d) t=%d maxE=%d\n", len(xs), cx, cy, dx, dy, t, maxE)
		}
		return 0, false
	}
	if os.Getenv("ICONFORGE_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "fitCornerArc: 样本数=%d 角=(%d,%d) dir=(%d,%d) t=%d maxE=%d\n", len(xs), cx, cy, dx, dy, t, maxE)
		for i := range xs {
			fmt.Fprintf(os.Stderr, "  pt=(%d,%d)\n", int(xs[i]), int(ys[i]))
		}
	}
	// 圆角弧的圆心必在角点出发的 45° 对角线方向上（dx==dy），用 A=B 约束做单变量最小二乘，
	// 比 3 变量一般圆方程更稳定（数据点都在对角线方向，3 变量欠约束）
	r, ok := leastSquaresCircleFixedCenter(xs, ys, dx, dy)
	if ok {
		// 物理合理性：圆心 C0 = dx*pad+dx*r 应在图片内；半径在合理范围
		if r < 2 || r > maxE {
			if os.Getenv("ICONFORGE_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "  reject: r=%d 超出 [2,%d]\n", r, maxE)
			}
			return 0, false
		}
		if os.Getenv("ICONFORGE_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "fitCornerArc: 拟合 r=%d\n", r)
		}
	}
	return r, ok
}

// findEdgeAlongRow 在 y=cy+dy*v 这一行上从角点 cx+dx 方向出发，
// 找到"第一次出现连续 3 像素为前景（颜色与背景差异 > 阈值）"的 x 位置作为边界。
func findEdgeAlongRow(img image.Image, bounds image.Rectangle, cx, cy, dx, dy, v int, bg color.RGBA) (int, bool) {
	w := bounds.Dx()
	h := bounds.Dy()
	py := cy + dy*v
	if py < 0 || py >= h {
		return 0, false
	}
	startX := cx + dx
	if startX < 0 {
		startX = 0
	}
	end := startX + dx*w/3 // 沿方向最多扫 1/3 边长
	if dx > 0 {
		if end > w {
			end = w
		}
	} else {
		if end < 0 {
			end = 0
		}
	}
	for x := startX; ; x += dx {
		if dx > 0 && x >= end || dx < 0 && x <= end {
			return 0, false
		}
		if x < 0 || x >= w {
			return 0, false
		}
		// 连续 3 像素都是前景
		if isFg(pixelAt(img, bounds, x, py), bg) &&
			x+dx >= 0 && x+dx < w && isFg(pixelAt(img, bounds, x+dx, py), bg) &&
			x+2*dx >= 0 && x+2*dx < w && isFg(pixelAt(img, bounds, x+2*dx, py), bg) {
			return x, true
		}
	}
}

// findEdgeAlongCol 在 x=cx+dx*u 这一列上找 y 边界，逻辑同 findEdgeAlongRow
func findEdgeAlongCol(img image.Image, bounds image.Rectangle, cx, cy, dx, dy, u int, bg color.RGBA) (int, bool) {
	w := bounds.Dx()
	h := bounds.Dy()
	px := cx + dx*u
	if px < 0 || px >= w {
		return 0, false
	}
	startY := cy + dy
	if startY < 0 {
		startY = 0
	}
	end := startY + dy*h/3
	if dy > 0 {
		if end > h {
			end = h
		}
	} else {
		if end < 0 {
			end = 0
		}
	}
	for y := startY; ; y += dy {
		if dy > 0 && y >= end || dy < 0 && y <= end {
			return 0, false
		}
		if y < 0 || y >= h {
			return 0, false
		}
		if isFg(pixelAt(img, bounds, px, y), bg) &&
			y+dy >= 0 && y+dy < h && isFg(pixelAt(img, bounds, px, y+dy), bg) &&
			y+2*dy >= 0 && y+2*dy < h && isFg(pixelAt(img, bounds, px, y+2*dy), bg) {
			return y, true
		}
	}
}

// leastSquaresCircle 对点集做一般圆方程 (x-A)²+(y-B)²=R² 的最小二乘拟合。
// 展开： x²+y² = 2Ax + 2By + (R²-A²-B²)，线性化为 M·[A,B,C]^T = b，克莱姆法则求解。
func leastSquaresCircle(xs, ys []float64, w, h int) (int, bool) {
	n := float64(len(xs))
	var sx, sy, sxx, syy, sxy float64
	var b0, b1, b2 float64
	for i := range xs {
		x, y := xs[i], ys[i]
		sx += x
		sy += y
		sxx += x * x
		syy += y * y
		sxy += x * y
		p := x*x + y*y
		b0 += x * p
		b1 += y * p
		b2 += p
	}
	// 圆方程 (x-A)²+(y-B)²=R² → x²+y² = 2A·x + 2B·y + (R²-A²-B²)
	// 设 c₀ = R²-A²-B²，对 A,B,c₀ 求偏导得线性方程组
	//   [2Σx²  2Σxy  Σx ] [A ]   [Σx·p]
	//   [2Σxy  2Σy²  Σy ] [B ] = [Σy·p]    (p = x²+y²)
	//   [Σx    Σy    n  ] [c₀]   [Σp    ]
	M := [3][3]float64{
		{2 * sxx, 2 * sxy, sx},
		{2 * sxy, 2 * syy, sy},
		{sx, sy, n},
	}
	B := [3]float64{b0, b1, b2}
	det := M[0][0]*(M[1][1]*M[2][2]-M[1][2]*M[2][1]) -
		M[0][1]*(M[1][0]*M[2][2]-M[1][2]*M[2][0]) +
		M[0][2]*(M[1][0]*M[2][1]-M[1][1]*M[2][0])
	debugReject := func(reason string) (int, bool) {
		if os.Getenv("ICONFORGE_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "  reject: %s\n", reason)
		}
		return 0, false
	}
	if math.Abs(det) < 1e-3 {
		return debugReject("det too small")
	}
	detA := B[0]*(M[1][1]*M[2][2]-M[1][2]*M[2][1]) -
		M[0][1]*(B[1]*M[2][2]-M[1][2]*B[2]) +
		M[0][2]*(B[1]*M[2][1]-M[1][1]*B[2])
	detB := M[0][0]*(B[1]*M[2][2]-M[1][2]*B[2]) -
		B[0]*(M[1][0]*M[2][2]-M[1][2]*M[2][0]) +
		M[0][2]*(M[1][0]*B[2]-B[1]*M[2][0])
	detC := M[0][0]*(M[1][1]*B[2]-B[1]*M[2][1]) -
		M[0][1]*(M[1][0]*B[2]-B[1]*M[2][0]) +
		B[0]*(M[1][0]*M[2][1]-M[1][1]*M[2][0])
	A := detA / det
	Bc := detB / det
	Cc := detC / det
	if os.Getenv("ICONFORGE_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "  center=(%.1f,%.1f) C=%.1f\n", A, Bc, Cc)
	}
	r2 := A*A + Bc*Bc + Cc
	if r2 <= 0.5 {
		return debugReject("r2 <= 0.5")
	}
	r := math.Sqrt(r2)
	if os.Getenv("ICONFORGE_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "  r=%.1f\n", r)
	}
	if A < -10 || A > float64(w)+10 || Bc < -10 || Bc > float64(h)+10 {
		return debugReject("center out of image")
	}
	if r < 4 || r > float64(min(w, h))/1.5 {
		return debugReject("r out of range")
	}
	return int(r + 0.5), true
}

// leastSquaresCircleFixedCenter 对每个角的样本做"圆心在角的对角线方向"的约束拟合。
// 旋转变换：把对角线方向作为新 u 轴（沿角点出发的 45° 射线），v 垂直于 u。
// 旋转角度：u = x·dx + y·dy（点积），v = -x·dy + y·dx（叉积）。
// 圆心在对角线上 ⇒ v=0 ⇒ (u-C0)²+v²=R² ⇒ u²+v² = 2C0·u + (R²-C0²)
// 对样本 (u, v) 计算 (u²+v²) 和 u，单变量最小二乘。
func leastSquaresCircleFixedCenter(xs, ys []float64, dx, dy int) (int, bool) {
	n := float64(len(xs))
	if n < 3 {
		return 0, false
	}
	var su, sp, suu, sup float64
	for i := range xs {
		// 旋转变换：u 沿对角线方向，v 垂直
		u := float64(xs[i])*float64(dx) + float64(ys[i])*float64(dy)
		_ = u
		v := -float64(xs[i])*float64(dy) + float64(ys[i])*float64(dx)
		_ = v
		p := float64(xs[i])*float64(xs[i]) + float64(ys[i])*float64(ys[i])
		su += u
		sp += p
		suu += u * u
		sup += u * p
	}
	denom := n*suu - su*su
	if math.Abs(denom) < 1e-3 {
		return 0, false
	}
	slope := (n*sup - su*sp) / denom     // = 2k
	intercept := (sp - slope*su) / n     // = R² - 2k²
	C0 := slope / 2
	r2 := intercept + 2*C0*C0
	if os.Getenv("ICONFORGE_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "  C0=%.1f r2=%.1f r=%.1f\n", C0, r2, math.Sqrt(max(r2, 0)))
	}
	if r2 <= 0.5 || C0 <= 0 {
		return 0, false
	}
	r := math.Sqrt(r2)
	return int(r + 0.5), true
}

// isFg 颜色差异大于阈值即视为前景
// 阈值 10：与 isBg(<=12) 配套，保留少量缓冲（12-15 是抗锯齿过渡区）
func isFg(c, bg color.RGBA) bool {
	return colorDistance(c, bg) > 10
}

// averageColor 采样一个区域的平均色
func averageColor(img image.Image, x, y, w, h int) color.RGBA {
	var sr, sg, sb, sa, n uint64
	bounds := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx < bounds.Min.X || xx >= bounds.Max.X || yy < bounds.Min.Y || yy >= bounds.Max.Y {
				continue
			}
			r, g, b, a := img.At(xx, yy).RGBA()
			sr += uint64(r >> 8)
			sg += uint64(g >> 8)
			sb += uint64(b >> 8)
			sa += uint64(a >> 8)
			n++
		}
	}
	if n == 0 {
		return color.RGBA{}
	}
	return color.RGBA{R: uint8(sr / n), G: uint8(sg / n), B: uint8(sb / n), A: uint8(sa / n)}
}

// colorDistance 两颜色的欧氏距离（0~441），双透明视为相同
func colorDistance(a, b color.RGBA) int {
	if a.A < 20 && b.A < 20 {
		return 0
	}
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return int(math.Sqrt(float64(dr*dr + dg*dg + db*db)))
}

// ===================== 圆角切割 =====================

// ApplyRoundedCorners 用圆角矩形蒙版切除四角（变透明），边缘做抗锯齿过渡
func ApplyRoundedCorners(img image.Image, radius int) *image.NRGBA {
	dst := imaging.Clone(img)
	if radius <= 0 {
		return dst
	}
	w := dst.Rect.Dx()
	h := dst.Rect.Dy()
	r := float64(radius)
	if r > float64(w)/2 {
		r = float64(w) / 2
	}
	if r > float64(h)/2 {
		r = float64(h) / 2
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := roundedRectSDF(float64(x)+0.5, float64(y)+0.5, float64(w), float64(h), r)
			if d <= -0.5 {
				continue // 完全在内部，保留
			}
			alpha := 0.5 - d // d=-0.5 -> 1.0（内）；d=0.5 -> 0.0（外）
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
			px := dst.NRGBAAt(x, y)
			px.A = uint8(float64(px.A) * alpha)
			dst.SetNRGBA(x, y, px)
		}
	}
	return dst
}

// roundedRectSDF 圆角矩形的带符号距离场（负=内部，正=外部）
func roundedRectSDF(px, py, w, h, r float64) float64 {
	hw, hh := w/2, h/2
	qx := math.Abs(px-hw) - (hw - r)
	qy := math.Abs(py-hh) - (hh - r)
	ox := math.Max(qx, 0)
	oy := math.Max(qy, 0)
	return math.Sqrt(ox*ox+oy*oy) + math.Min(math.Max(qx, qy), 0) - r
}

// ===================== 正方形裁剪与缩放 =====================

// CropSquare 按裁剪框（原图坐标）裁出正方形区域
func CropSquare(img image.Image, x, y, size int) *image.NRGBA {
	return imaging.Crop(img, image.Rect(x, y, x+size, y+size))
}

// ResizeHighQuality 高质量缩放（Lanczos）
func ResizeHighQuality(img image.Image, size int) *image.NRGBA {
	return imaging.Resize(img, size, size, imaging.Lanczos)
}

// ===================== ICO 编码 =====================

// EncodeICO 将多张图打包成单个 ICO 文件。
// 小尺寸（<=128）用 BMP 格式条目（ICO 原始标准，所有查看器都支持），
// 256 用 PNG 压缩（官方为大图标设计，体积小）。
func EncodeICO(images []image.Image) ([]byte, error) {
	var buf bytes.Buffer

	count := len(images)
	// ICO 文件头（6 字节）：保留0 + 类型1(ico) + 数量
	buf.Write([]byte{0, 0, 1, 0, byte(count), byte(count >> 8)})

	offset := 6 + 16*count
	var entryData [][]byte
	for _, img := range images {
		var data []byte
		var err error
		if img.Bounds().Dx() >= 256 {
			var pngBuf bytes.Buffer
			if err = png.Encode(&pngBuf, img); err != nil {
				return nil, err
			}
			data = pngBuf.Bytes()
		} else {
			data, err = encodeICOEntryBMP(img)
			if err != nil {
				return nil, err
			}
		}
		entryData = append(entryData, data)

		b := img.Bounds().Dx()
		bW := byte(b)
		if b >= 256 {
			bW = 0 // ICO 规范：256 用 0 表示
		}
		buf.WriteByte(bW) // 宽
		buf.WriteByte(bW) // 高
		buf.WriteByte(0)  // 调色板数
		buf.WriteByte(0)  // 保留
		buf.WriteByte(1)  // 色彩平面数
		buf.WriteByte(0)
		buf.WriteByte(32) // 位深
		buf.WriteByte(0)
		writeLE32(&buf, len(data)) // 数据长度
		writeLE32(&buf, offset)    // 数据偏移
		offset += len(data)
	}

	for _, data := range entryData {
		buf.Write(data)
	}
	return buf.Bytes(), nil
}

// encodeICOEntryBMP 将图像编码为 ICO 内的 32bpp BMP 条目（BITMAPINFOHEADER + BGRA 像素 + AND 掩码）
func encodeICOEntryBMP(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	// BITMAPINFOHEADER（40 字节），注意 biHeight = 2*h（XOR 图 + AND 掩码各占一份高度）
	header := make([]byte, 40)
	binary.LittleEndian.PutUint32(header[0:], 40)
	binary.LittleEndian.PutUint32(header[4:], uint32(w))
	binary.LittleEndian.PutUint32(header[8:], uint32(h*2))
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], 32)
	// biCompression = BI_RGB(0)，其余字段 0

	data := make([]byte, 0, 40+w*h*4+((w+7)/8+3)/4*4*h)
	data = append(data, header...)

	// 像素数据：BGRA，自底向上
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x, b.Min.Y+y)
			r, g, bl, a := c.RGBA()
			data = append(data, byte(bl>>8), byte(g>>8), byte(r>>8), byte(a>>8))
		}
	}

	// AND 掩码：每行 (w+7)/8 字节并按 4 字节对齐，32bpp 有 alpha 时全 0 即可
	rowBytes := (w + 7) / 8
	if rem := rowBytes % 4; rem != 0 {
		rowBytes += 4 - rem
	}
	data = append(data, make([]byte, rowBytes*h)...)

	return data, nil
}

func writeLE32(buf *bytes.Buffer, v int) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 24))
}

// ===================== 处理管线 =====================

// BuildIconPipeline 完整处理管线：正方形裁剪 -> 圆角切割 -> 各尺寸缩放 -> ICO 编码
func BuildIconPipeline(src image.Image, cropX, cropY, cropSize, cornerRadius int, sizes []int) ([]byte, error) {
	// 裁剪框越界保护
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if cropSize <= 0 || cropSize > w || cropSize > h {
		cropSize = int(math.Min(float64(w), float64(h)))
		cropX = (w - cropSize) / 2
		cropY = (h - cropSize) / 2
	}
	if cropX < 0 {
		cropX = 0
	}
	if cropY < 0 {
		cropY = 0
	}
	if cropX+cropSize > w {
		cropX = w - cropSize
	}
	if cropY+cropSize > h {
		cropY = h - cropSize
	}

	square := CropSquare(src, cropX, cropY, cropSize)
	rounded := ApplyRoundedCorners(square, cornerRadius)

	var iconImages []image.Image
	for _, s := range sizes {
		resized := ResizeHighQuality(rounded, s)
		// 小尺寸下圆角边缘容易残留杂边，缩放后再按比例重切一次
		if cornerRadius > 0 && s <= 48 {
			r := cornerRadius * s / cropSize
			if r > 0 {
				resized = ApplyRoundedCorners(resized, r)
			}
		}
		iconImages = append(iconImages, resized)
	}
	return EncodeICO(iconImages)
}
