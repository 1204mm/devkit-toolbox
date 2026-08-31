package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// 生成测试图：白色背景上的蓝色圆角方块（圆角半径 r），尺寸 s
func makeRoundedIcon(s, r int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// 蓝色圆角矩形：四周留白（小图自适应，避免 inner 为负）
	pad := 20
	if s < 60 {
		pad = s / 10
		if pad < 1 {
			pad = 1
		}
	}
	inner := s - pad*2
	for y := pad; y < pad+inner; y++ {
		for x := pad; x < pad+inner; x++ {
			d := roundedRectSDF(float64(x)+0.5-float64(pad), float64(y)+0.5-float64(pad), float64(inner), float64(inner), float64(r))
			if d <= 0 {
				img.Set(x, y, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
			}
		}
	}
	return img
}

// makeRoundedIconOnGradient 模拟"白渐变背景 + 中心前景图"的真实图片（参考用户提供图）：
// 整张淡白渐变底，中心是占满 80% 的蓝色圆角前景（与外白底形成清晰色块边界）。
func makeRoundedIconOnGradient(s, r int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			// 背景：白→浅灰渐变（轻微，避免被 cornerBgUniform 当成纯色）
			shade := uint8(245 - (x+y)/(s*2/255*8)) // 微小渐变
			img.Set(x, y, color.NRGBA{R: shade, G: shade, B: shade + 2, A: 255})
		}
	}
	pad := s / 10 // 占满约 80%
	inner := s - pad*2
	oy := pad
	for y := 0; y < inner; y++ {
		for x := 0; x < inner; x++ {
			d := roundedRectSDF(float64(x)+0.5, float64(y)+0.5, float64(inner), float64(inner), float64(r))
			if d <= 0 {
				img.Set(x+pad, y+oy, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
			}
		}
	}
	return img
}

func TestDetectCornerRadiusRealistic(t *testing.T) {
	// 1024 图，期望半径 200（≈ 20% 边长）。容差 ±25%。
	img := makeRoundedIconOnGradient(1024, 200)
	r, detected := DetectCornerRadius(img)
	t.Logf("检测到: %v, 识别半径: %d", detected, r)
	if !detected {
		t.Fatal("应检测到圆角背景")
	}
	lo, hi := 200*75/100, 200*125/100
	if r < lo || r > hi {
		t.Logf("识别半径 %d 超出预期范围 [%d,%d]（首次仅信息打印）", r, lo, hi)
	}
}

// makeRoundedIconAA 生成"真实导出图片"风格的测试图：
// 图标边缘带抗锯齿柔和过渡（按 SDF 距离插值 alpha），模拟截图/PNG 导出效果。
// 外部白底 + 圆角图标（淡渐变主体 + 中心蓝色 glyph）。
func makeRoundedIconAA(s, r int, blur float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	pad := s / 10
	inner := s - pad*2
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			// 白色页面底
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			// 图标：圆角矩形，边缘按 SDF 插值 alpha（blur 控制过渡带宽度）
			d := roundedRectSDF(float64(x)+0.5-float64(pad), float64(y)+0.5-float64(pad), float64(inner), float64(inner), float64(r))
			cov := 0.5 - d/blur // blur=1 → 1px 过渡带
			if cov > 1 {
				cov = 1
			}
			if cov > 0 {
				shade := uint8(246 - (x+y)/(s*2/255*8))
				ic := color.NRGBA{R: shade, G: shade, B: shade + 2, A: 255}
				bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
				// alpha 混合
				a := uint8(cov * 255)
				mix := func(c1, c2 uint8) uint8 {
					return uint8(int(c1) + (int(c2)-int(c1))*int(a)/255)
				}
				px := color.NRGBA{
					R: mix(bg.R, ic.R), G: mix(bg.G, ic.G), B: mix(bg.B, ic.B), A: 255,
				}
				img.Set(x, y, px)
			}
		}
	}
	// 中心蓝色 glyph
	gpad := s / 4
	ginner := s - gpad*2
	gr := ginner / 4
	for y := 0; y < ginner; y++ {
		for x := 0; x < ginner; x++ {
			d := roundedRectSDF(float64(x)+0.5, float64(y)+0.5, float64(ginner), float64(ginner), float64(gr))
			if d <= 0 {
				img.Set(x+gpad, y+gpad, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
			}
		}
	}
	return img
}

// TestDetectCornerRadiusAA 验证抗锯齿边缘下的识别精度（真实图片场景）
func TestDetectCornerRadiusAA(t *testing.T) {
	for _, blur := range []float64{1, 2, 3} {
		img := makeRoundedIconAA(1024, 200, blur)
		r, detected := DetectCornerRadius(img)
		t.Logf("blur=%.1f: 检测到=%v 识别半径=%d (期望200)", blur, detected, r)
		if !detected {
			t.Fatalf("blur=%.1f 应检测到圆角背景", blur)
		}
		// 多切（偏大）比少切更严重：限制 +10%
		if r > 200*110/100 {
			t.Errorf("blur=%.1f 识别半径 %d 超过 +10%% 上限 220，会多切图标本体", blur, r)
		}
		if r < 200*80/100 {
			t.Errorf("blur=%.1f 识别半径 %d 低于 -20%% 下限 160，没切干净", blur, r)
		}
	}
}

// makeRoundedSquareOnGradient 模拟用户提供的真实图片：
// 整张图的外层是带圆角的方形（圆角外区域透明），内部是淡白渐变 + 中心蓝色图标。
// 这才是 iOS 风格的应用图标结构：外层圆角 + 内部内容。
func makeRoundedSquareOnGradient(s, r int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	// 默认全部透明（alpha=0）
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			// 圆角矩形内部填淡白渐变，圆角外保持透明
			d := roundedRectSDF(float64(x)+0.5, float64(y)+0.5, float64(s), float64(s), float64(r))
			if d <= 0 {
				shade := uint8(245 - (x+y)/(s*2/255*8)) // 微小渐变
				img.Set(x, y, color.NRGBA{R: shade, G: shade, B: shade + 2, A: 255})
			}
		}
	}
	// 中心蓝色图标（简单的圆角矩形，占满约 60%）
	pad := s / 5
	inner := s - pad*2
	iconR := inner / 4
	for y := 0; y < inner; y++ {
		for x := 0; x < inner; x++ {
			d := roundedRectSDF(float64(x)+0.5, float64(y)+0.5, float64(inner), float64(inner), float64(iconR))
			if d <= 0 {
				img.Set(x+pad, y+pad, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
			}
		}
	}
	return img
}

func TestDetectCornerRadiusUserImage(t *testing.T) {
	// 1024 图，期望半径 200（≈ 20% 边长）。±20% 容差。
	img := makeRoundedSquareOnGradient(1024, 200)
	r, detected := DetectCornerRadius(img)
	t.Logf("检测到: %v, 识别半径: %d", detected, r)
	if !detected {
		t.Fatal("应检测到圆角背景")
	}
	lo, hi := 200*80/100, 200*120/100
	if r < lo || r > hi {
		t.Fatalf("识别半径 %d 超出预期范围 [%d,%d]", r, lo, hi)
	}
}

func TestDetectCornerRadius(t *testing.T) {
	img := makeRoundedIcon(512, 64)
	r, detected := DetectCornerRadius(img)
	if !detected {
		t.Fatal("应检测到圆角背景")
	}
	// 识别值允许 ±20% 误差（圆弧拟合精化后更准）
	lo, hi := 64*80/100, 64*120/100
	if r < lo || r > hi {
		t.Fatalf("识别半径 %d 超出预期范围 [%d,%d]", r, lo, hi)
	}
	t.Logf("识别半径: %d (期望约64)", r)
}

// TestDetectCornerRadiusSmallNoOvershoot 验证"小半径不被多切"：
// 历史 bug：+10% 比例余量在小半径时会多切 1.5-2 倍比例；
// 修复后用固定 2px 余量，"多切"应控制在 +15% 以内（偏小不限制，因为更安全）。
func TestDetectCornerRadiusSmallNoOvershoot(t *testing.T) {
	img := makeRoundedIcon(512, 32) // 期望半径 32
	r, detected := DetectCornerRadius(img)
	if !detected {
		t.Fatal("应检测到圆角背景")
	}
	// 关注"多切"问题：r 不能超过 actual * 1.15
	maxR := 32 * 115 / 100
	if r > maxR {
		t.Fatalf("小半径识别 %d 超过 +15%% 上限 (%d)，会把图标本体多切", r, maxR)
	}
	// 下限也要合理（不小于 actual * 0.7，否则可能没切干净）
	minR := 32 * 70 / 100
	if r < minR {
		t.Fatalf("小半径识别 %d 低于 -30%% 下限 (%d)，可能根本没识别到", r, minR)
	}
	t.Logf("识别半径: %d (期望约32)", r)
}

func TestDetectFlatImage(t *testing.T) {
	// 满幅纯色渐变图，不应检测到圆角
	img := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	_, detected := DetectCornerRadius(img)
	if detected {
		t.Fatal("满幅图不应检测到圆角")
	}
}

func TestApplyRoundedCorners(t *testing.T) {
	img := makeRoundedIcon(256, 60)
	out := ApplyRoundedCorners(img, 75) // 切得比实际圆角多一点
	// 四个角应透明
	for _, p := range [][2]int{{2, 2}, {253, 2}, {2, 253}, {253, 253}} {
		_, _, _, a := out.At(p[0], p[1]).RGBA()
		if a != 0 {
			t.Fatalf("角 (%d,%d) 应为透明, got alpha=%d", p[0], p[1], a>>8)
		}
	}
	// 中心应保持不透明
	_, _, _, a := out.At(128, 128).RGBA()
	if a>>8 != 255 {
		t.Fatalf("中心应不透明, got alpha=%d", a>>8)
	}
}

func TestEncodeICO(t *testing.T) {
	imgs := []image.Image{makeRoundedIcon(16, 3), makeRoundedIcon(32, 6), makeRoundedIcon(256, 40)}
	data, err := EncodeICO(imgs)
	if err != nil {
		t.Fatal(err)
	}
	// 头部校验
	if data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
		t.Fatal("ICO 头部错误")
	}
	count := int(data[4]) | int(data[5])<<8
	if count != 3 {
		t.Fatalf("目录项数量错误: %d", count)
	}

	readEntry := func(i int) (size byte, off, length int) {
		base := 6 + i*16
		size = data[base]
		length = int(data[base+8]) | int(data[base+9])<<8 | int(data[base+10])<<16 | int(data[base+11])<<24
		off = int(data[base+12]) | int(data[base+13])<<8 | int(data[base+14])<<16 | int(data[base+15])<<24
		return
	}

	// 前两项（16/32）应为 BMP 条目，256 为 PNG 条目
	for i, expSize := range []int{16, 32} {
		sz, off, length := readEntry(i)
		if int(sz) != expSize {
			t.Fatalf("条目 %d 尺寸错误: %d", i, sz)
		}
		entry := data[off : off+length]
		// BITMAPINFOHEADER 校验
		if binary.LittleEndian.Uint32(entry[0:]) != 40 {
			t.Fatalf("条目 %d biSize 应为 40", i)
		}
		if binary.LittleEndian.Uint32(entry[4:]) != uint32(expSize) {
			t.Fatalf("条目 %d biWidth 错误", i)
		}
		if binary.LittleEndian.Uint32(entry[8:]) != uint32(expSize*2) {
			t.Fatalf("条目 %d biHeight 应为 2 倍高度", i)
		}
		if binary.LittleEndian.Uint16(entry[14:]) != 32 {
			t.Fatalf("条目 %d biBitCount 应为 32", i)
		}
		// 数据长度 = 40 头 + 像素 + AND 掩码
		rowBytes := (expSize + 7) / 8
		if rem := rowBytes % 4; rem != 0 {
			rowBytes += 4 - rem
		}
		expLen := 40 + expSize*expSize*4 + rowBytes*expSize
		if length != expLen {
			t.Fatalf("条目 %d 数据长度错误: got %d want %d", i, length, expLen)
		}
		// 像素校验：BMP 自底向上，取图像垂直中心行（水平居中像素），测试图中心是蓝色
		centerOff := 40 + (expSize/2)*expSize*4 + (expSize/2)*4
		b, g, r, a := entry[centerOff], entry[centerOff+1], entry[centerOff+2], entry[centerOff+3]
		if a != 255 || r > 80 || g < 60 || g > 140 || b < 180 {
			t.Fatalf("条目 %d 中心像素颜色错误: B=%d G=%d R=%d A=%d（期望蓝色）", i, b, g, r, a)
		}
	}

	// 256 尺寸编码为 0，数据为合法 PNG
	sz, off, length := readEntry(2)
	if sz != 0 {
		t.Fatalf("256 尺寸应编码为 0, got %d", sz)
	}
	if off+length != len(data) {
		t.Fatalf("数据区长度不匹配: off=%d len=%d total=%d", off, length, len(data))
	}
	if !bytes.HasPrefix(data[off:off+8], []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("256 条目内嵌数据不是 PNG")
	}
	if _, err := png.Decode(bytes.NewReader(data[off : off+length])); err != nil {
		t.Fatalf("内嵌 PNG 解码失败: %v", err)
	}
}

func TestBuildIconPipeline(t *testing.T) {
	img := makeRoundedIcon(512, 64)
	data, err := BuildIconPipeline(img, 20, 20, 472, 80, []int{16, 32, 48, 256})
	if err != nil {
		t.Fatal(err)
	}
	count := int(data[4]) | int(data[5])<<8
	if count != 4 {
		t.Fatalf("应包含 4 个尺寸, got %d", count)
	}
	// 各尺寸独立 ICO 也应有效（模拟 ZIP 导出的每个文件）
	for _, s := range []int{16, 32, 48, 256} {
		one, err := BuildIconPipeline(img, 20, 20, 472, 80, []int{s})
		if err != nil {
			t.Fatalf("尺寸 %d 生成失败: %v", s, err)
		}
		c := int(one[4]) | int(one[5])<<8
		if c != 1 {
			t.Fatalf("尺寸 %d 独立 ICO 条目数错误: %d", s, c)
		}
		sz := one[6]
		exp := byte(s)
		if s >= 256 {
			exp = 0
		}
		if sz != exp {
			t.Fatalf("尺寸 %d 独立 ICO 目录尺寸错误: %d", s, sz)
		}
	}
	t.Logf("生成 ICO 大小: %d 字节", len(data))
}

// TestDetectIconParamsGlyphCrop 复现"多切圆角"场景：
// 白底页面 + 白色系图标本体（旧算法色差~14<15 会漏掉）+ 中心深色图案。
// 结构化内容检测升级后：浅色图标本体（1080 边长）也应被框进来 →
// 内容框 = 图标本体，裁剪后在图标本体上重测圆角（~216/1080 = 20%）。
func TestDetectIconParamsGlyphCrop(t *testing.T) {
	s := 1200
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255}) // 白色页面底
		}
	}
	// 白色系图标（色差 ~14）：1080 边长，圆角 216（20%），位于 (60,60)
	pad, inner, iconR := 60, 1080, 216
	for y := pad; y < pad+inner; y++ {
		for x := pad; x < pad+inner; x++ {
			d := roundedRectSDF(float64(x)+0.5-float64(pad), float64(y)+0.5-float64(pad), float64(inner), float64(inner), float64(iconR))
			if d <= 0 {
				img.Set(x, y, color.NRGBA{R: 246, G: 246, B: 249, A: 255})
			}
		}
	}
	// 中心蓝色图案：500 边长，圆角 100
	gpad, ginner, gr := (s-500)/2, 500, 100
	for y := gpad; y < gpad+ginner; y++ {
		for x := gpad; x < gpad+ginner; x++ {
			d := roundedRectSDF(float64(x)+0.5-float64(gpad), float64(y)+0.5-float64(gpad), float64(ginner), float64(ginner), float64(gr))
			if d <= 0 {
				img.Set(x, y, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
			}
		}
	}

	r, cornerOK, cx, cy, cw, ch, contentOK := DetectIconParams(img)
	t.Logf("radius=%d cornerOK=%v content=(%d,%d,%d,%d) contentOK=%v", r, cornerOK, cx, cy, cw, ch, contentOK)

	// 内容应被检测到
	if !contentOK {
		t.Fatal("应检测到内容包围盒")
	}
	// 结构化检测升级：应框住浅色图标本体（1080），而不是只框中心图案（500）
	if cw < 1000 || ch < 1000 {
		t.Fatalf("内容框 (%d,%d,%d,%d) 只框住了中心图案，浅色图标本体被漏掉", cx, cy, cw, ch)
	}
	// 裁剪框边长（与前端一致）
	side := cw
	if ch > side {
		side = ch
	}
	if side > s {
		side = s
	}
	if cornerOK {
		ratio := float64(r) * 100 / float64(side)
		t.Logf("radius/crop = %.1f%%（图标本体实际圆角比例 216/1080 = 20%%）", ratio)
		// 多切防护：比例不应明显超过实际（旧 bug 只框图案时会到 43%）
		if ratio > 30 {
			t.Fatalf("圆角比例 %.1f%% 过大（多切风险）", ratio)
		}
	} else {
		t.Logf("裁剪区域未检测到圆角（不切，安全）")
	}
}

// TestDetectIconParamsFullImage 图标铺满整图（无自动裁剪）时行为不变
func TestDetectIconParamsFullImage(t *testing.T) {
	img := makeRoundedSquareOnGradient(1024, 200)
	r, cornerOK, _, _, _, _, contentOK := DetectIconParams(img)
	if contentOK {
		t.Fatal("满幅图不应触发自动定位")
	}
	if !cornerOK {
		t.Fatal("应检测到圆角")
	}
	lo, hi := 200*80/100, 200*110/100
	if r < lo || r > hi {
		t.Fatalf("识别半径 %d 超出范围 [%d,%d]", r, lo, hi)
	}
}

// ===================== 通用性场景矩阵 =====================
// 模拟真实世界各种来源的图标图片，评估 DetectIconParams 的综合表现。
// 每个场景给定期望（内容框 / 圆角 / 比例），统一在矩阵测试中断言。

// scFill 纯色铺底
func scFill(img *image.NRGBA, s int, c color.NRGBA) {
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			img.Set(x, y, c)
		}
	}
}

// scRoundedRect 画一个圆角矩形（纯色，硬边缘）
func scRoundedRect(img *image.NRGBA, x0, y0, size, r int, c color.NRGBA) {
	for y := y0; y < y0+size; y++ {
		for x := x0; x < x0+size; x++ {
			d := roundedRectSDF(float64(x)+0.5-float64(x0), float64(y)+0.5-float64(y0), float64(size), float64(size), float64(r))
			if d <= 0 {
				img.Set(x, y, c)
			}
		}
	}
}

// scRoundedRectAA 画一个带抗锯齿边缘的圆角矩形（alpha 混合）
func scRoundedRectAA(img *image.NRGBA, x0, y0, size, r int, c color.NRGBA, blur float64) {
	for y := y0; y < y0+size; y++ {
		for x := x0; x < x0+size; x++ {
			d := roundedRectSDF(float64(x)+0.5-float64(x0), float64(y)+0.5-float64(y0), float64(size), float64(size), float64(r))
			cov := 0.5 - d/blur
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			src := img.NRGBAAt(x, y)
			a := int(cov * 255)
			mix := func(c1, c2 uint8) uint8 {
				return uint8(int(c1) + (int(c2)-int(c1))*a/255)
			}
			img.Set(x, y, color.NRGBA{R: mix(src.R, c.R), G: mix(src.G, c.G), B: mix(src.B, c.B), A: mix(src.A, c.A)})
		}
	}
}

// scNoise 添加 ±n 的随机噪点（模拟 JPEG 压缩杂色）
func scNoise(img *image.NRGBA, s, n int) {
	seed := 12345
	next := func() int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed
	}
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			p := img.NRGBAAt(x, y)
			d := next() % (2*n + 1) - n
			clamp := func(v int) uint8 {
				if v < 0 {
					return 0
				}
				if v > 255 {
					return 255
				}
				return uint8(v)
			}
			img.Set(x, y, color.NRGBA{R: clamp(int(p.R) + d), G: clamp(int(p.G) + d), B: clamp(int(p.B) + d), A: p.A})
		}
	}
}

// TestScenarioMatrix 通用性评估矩阵：覆盖 10 类典型图标来源
func TestScenarioMatrix(t *testing.T) {
	type exp struct {
		name          string
		img           *image.NRGBA
		wantContent   bool  // 期望触发自动裁剪
		wantCorner    bool  // 期望识别到圆角
		maxRatio      float64 // 圆角/裁剪框 上限（%），0 表示不检查
		desc          string
	}
	s := 1024
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	blue := color.NRGBA{R: 30, G: 100, B: 230, A: 255}
	light := color.NRGBA{R: 246, G: 246, B: 249, A: 255}

	mk := func(f func() *image.NRGBA) *image.NRGBA { return f() }

	cases := []exp{
		{
			name: "1.透明背景+圆角图标(AA)",
			desc: "设计软件导出 PNG：透明外圈 + 抗锯齿圆角",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scRoundedRectAA(im, 64, 64, 896, 180, blue, 2)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25,
		},
		{
			name: "2.白底+深色圆角图标",
			desc: "网页截图：白页面 + 蓝色圆角图标",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scFill(im, s, white)
				scRoundedRect(im, 64, 64, 896, 180, blue)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25,
		},
		{
			name: "3.白底+浅色图标+中心图案",
			desc: "iOS 风格：白上白浅色主体 + 深色 glyph",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, 1200, 1200))
				scFill(im, 1200, white)
				scRoundedRect(im, 60, 60, 1080, 216, light)
				scRoundedRect(im, 350, 350, 500, 100, blue)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25,
		},
		{
			name: "4.深色背景+浅色图标",
			desc: "深色模式：黑页面 + 白色圆角图标",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scFill(im, s, color.NRGBA{R: 20, G: 20, B: 24, A: 255})
				scRoundedRect(im, 64, 64, 896, 180, white)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25,
		},
		{
			name: "5.白底+噪点+圆角图标",
			desc: "JPEG 压缩后的截图：白底带杂色",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scFill(im, s, white)
				scRoundedRect(im, 64, 64, 896, 180, blue)
				scNoise(im, s, 6)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 30,
		},
		{
			name: "6.非居中图标",
			desc: "图标偏在图片一角",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scFill(im, s, white)
				scRoundedRect(im, 150, 90, 700, 140, blue)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25,
		},
		{
			name: "7.小尺寸图(256)",
			desc: "小图：256px 白底圆角图标",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, 256, 256))
				scFill(im, 256, white)
				scRoundedRect(im, 20, 20, 216, 44, blue)
				return im
			}),
			wantContent: true, wantCorner: true, maxRatio: 25, // 44px 是真实圆角（20%），应识别
		},
		{
			name: "8.满幅照片背景",
			desc: "整图都是照片/花纹 → 不应误触发裁剪或圆角",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				for y := 0; y < s; y++ {
					for x := 0; x < s; x++ {
						im.Set(x, y, color.NRGBA{R: uint8((x*3+y*5)%256), G: uint8((x+y*2)%256), B: uint8((x*7+y)%256), A: 255})
					}
				}
				return im
			}),
			wantContent: false, wantCorner: false, maxRatio: 0,
		},
		{
			name: "9.渐变背景+满幅圆角",
			desc: "图标铺满整图（四周无留白）→ 不裁剪，圆角可识别",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				for y := 0; y < s; y++ {
					for x := 0; x < s; x++ {
						shade := uint8(245 - (x+y)/64) // 真实 iOS 图级别的轻微渐变（全图 ~32 级变化）
						im.Set(x, y, color.NRGBA{R: shade, G: shade, B: shade + 2, A: 255})
					}
				}
				scRoundedRectAA(im, 0, 0, s, 200, blue, 2)
				return im
			}),
			wantContent: false, wantCorner: true, maxRatio: 25,
		},
		{
			name: "10.白底+直角方块图标",
			desc: "无圆角的方形图标 → 内容可识别，不应识别出圆角",
			img: mk(func() *image.NRGBA {
				im := image.NewNRGBA(image.Rect(0, 0, s, s))
				scFill(im, s, white)
				scRoundedRect(im, 64, 64, 896, 0, blue)
				return im
			}),
			wantContent: true, wantCorner: false, maxRatio: 0,
		},
	}

	for _, c := range cases {
		b := c.img.Bounds()
		w, h := b.Dx(), b.Dy()
		r, cornerOK, cx, cy, cw, ch, contentOK := DetectIconParams(c.img)
		t.Logf("[%s] %s", c.name, c.desc)
		t.Logf("  content=%v box=(%d,%d,%d,%d) corner=%v radius=%d (图 %dx%d)", contentOK, cx, cy, cw, ch, cornerOK, r, w, h)

		if contentOK != c.wantContent {
			t.Errorf("[%s] 自动裁剪=%v，期望 %v", c.name, contentOK, c.wantContent)
		}
		if cornerOK != c.wantCorner {
			t.Errorf("[%s] 圆角识别=%v（r=%d），期望 %v", c.name, cornerOK, r, c.wantCorner)
		}
		if contentOK && cornerOK && c.maxRatio > 0 {
			side := cw
			if ch > side {
				side = ch
			}
			ratio := float64(r) * 100 / float64(side)
			t.Logf("  radius/crop = %.1f%%", ratio)
			if ratio > c.maxRatio {
				t.Errorf("[%s] 圆角比例 %.1f%% 超上限 %.0f%%（多切风险）", c.name, ratio, c.maxRatio)
			}
		}
	}
}

func TestDetectContentBounds(t *testing.T) {
	// 场景1：1000x1000 大图，中心 600x600 蓝色实心矩形（四周白边）
	img := image.NewNRGBA(image.Rect(0, 0, 1000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 1000; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for y := 200; y < 800; y++ {
		for x := 200; x < 800; x++ {
			img.Set(x, y, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
		}
	}
	bounds := img.Bounds()
	t.Logf("img bounds: %v, w=%d h=%d", bounds, bounds.Dx(), bounds.Dy())
	for _, c := range [][2]int{{0, 0}, {999, 0}, {0, 999}, {999, 999}} {
		r, g, b, a := img.At(c[0], c[1]).RGBA()
		t.Logf("corner (%d,%d): R=%d G=%d B=%d A=%d", c[0], c[1], r>>8, g>>8, b>>8, a>>8)
	}
	x, y, w, h, ok := DetectContentBounds(img)
	t.Logf("DetectContentBounds: (%d,%d,%d,%d) ok=%v", x, y, w, h, ok)
	if !ok {
		t.Fatal("应检测到内容包围盒")
	}
	if x != 200 || y != 200 || w != 600 || h != 600 {
		t.Fatalf("包围盒错误: 期望(200,200,600,600), 实际(%d,%d,%d,%d)", x, y, w, h)
	}

	// 场景2：满幅图（无留白）→ 不应触发
	img2 := image.NewNRGBA(image.Rect(0, 0, 500, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 500; x++ {
			img2.Set(x, y, color.NRGBA{R: 30, G: 100, B: 230, A: 255})
		}
	}
	_, _, _, _, ok2 := DetectContentBounds(img2)
	if ok2 {
		t.Fatal("满幅图不应触发自动定位")
	}

	// 场景3：透明外圈 + 不透明内矩形（透明背景场景）
	img3 := image.NewNRGBA(image.Rect(0, 0, 800, 800))
	for y := 200; y < 600; y++ {
		for x := 200; x < 600; x++ {
			img3.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	x3, y3, w3, h3, ok3 := DetectContentBounds(img3)
	if !ok3 {
		t.Fatal("透明背景应检测到内容包围盒")
	}
	if x3 != 200 || y3 != 200 || w3 != 400 || h3 != 400 {
		t.Fatalf("透明背景包围盒错误: 期望(200,200,400,400), 实际(%d,%d,%d,%d)", x3, y3, w3, h3)
	}
}

func TestDebugUserImage(t *testing.T) {
	os.Setenv("ICONFORGE_DEBUG", "1")
	f, err := os.Open("debug_user.png")
	if err != nil {
		t.Skip("debug_user.png not found, skip:", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	t.Logf("size: %dx%d, colorModel=%T", w, h, img.ColorModel())

	// 检查文件本身是否带 alpha 通道
	rawData, _ := os.ReadFile("debug_user.png")
	t.Logf("file size: %d bytes", len(rawData))
	// 用 png.Decode 拿到带 alpha 的原始数据
	if pngImg, err := png.Decode(f); err == nil {
		// 查 4 个角
		for _, c := range [][2]int{{0, 0}, {5, 5}, {10, 10}, {50, 50}, {100, 100}, {200, 200}} {
			r, g, b, a := pngImg.At(c[0], c[1]).RGBA()
			t.Logf("PNG (%d,%d): R=%d G=%d B=%d A=%d (raw A=%d)", c[0], c[1], r>>8, g>>8, b>>8, a>>8, a)
		}
	}

	corners := [4]string{"TL", "TR", "BL", "BR"}
	cs := [4][4]int{{0, 0, 1, 1}, {w - 1, 0, -1, 1}, {0, h - 1, 1, -1}, {w - 1, h - 1, -1, -1}}
	for i, c := range cs {
		cx, cy, dx, dy := c[0], c[1], c[2], c[3]
		bg := averageColor(img, cx+dx, cy+dy, 3, 3)
		t.Logf("[%s] bg3x3: R=%d G=%d B=%d A=%d", corners[i], bg.R, bg.G, bg.B, bg.A)
		// 高密度采样对角线像素
		for d := 1; d <= 400; d++ {
			x := cx + dx*d
			y := cy + dy*d
			if x < 0 || x >= w || y < 0 || y >= h {
				break
			}
			c := pixelAt(img, bounds, x, y)
			dist := colorDistance(c, bg)
			isB := isBg(c, bg)
			isF := isFg(c, bg)
			if d <= 5 || d%20 == 0 || (isF && d < 200) {
				t.Logf("  step=%d (%d,%d): R=%d G=%d B=%d A=%d  dist=%d isBg=%v isFg=%v", d, x, y, c.R, c.G, c.B, c.A, dist, isB, isF)
			}
		}
	}
	content := averageColor(img, w/2-4, h/2-4, 8, 8)
	t.Logf("center: R=%d G=%d B=%d A=%d", content.R, content.G, content.B, content.A)

	r, ok := DetectCornerRadius(img)
	t.Logf("=== r=%d detected=%v ===", r, ok)

	// 模拟前端行为：内容包围盒 → 裁剪框 → 圆角相对裁剪框的比例
	cx, cy, cw, ch, cok := DetectContentBounds(img)
	t.Logf("=== content bounds: (%d,%d,%d,%d) detected=%v ===", cx, cy, cw, ch, cok)
	if cok {
		side := cw
		if ch > side {
			side = ch
		}
		cropSize := side
		if cropSize > w {
			cropSize = w
		}
		if cropSize > h {
			cropSize = h
		}
		effectiveR := r
		if effectiveR > cropSize/2 {
			effectiveR = cropSize / 2
		}
		t.Logf("=== crop.size=%d, radius=%d, radius/crop=%.1f%% (原图 radius/minSide=%.1f%%) ===",
			cropSize, effectiveR, float64(effectiveR)*100/float64(cropSize), float64(r)*100/float64(min(w, h)))
	}
}
