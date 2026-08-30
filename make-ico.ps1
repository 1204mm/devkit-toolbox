Add-Type -AssemblyName System.Drawing

$src = 'icon-1024-rounded.png'
$outPath = 'build\windows\icon.ico'

function Make-Resized {
    param([int]$size)
    $img = [System.Drawing.Image]::FromFile($src)
    $dstRect = New-Object System.Drawing.Rectangle(0, 0, $size, $size)
    $srcRect = New-Object System.Drawing.Rectangle(0, 0, $img.Width, $img.Height)
    $bmp = New-Object System.Drawing.Bitmap($size, $size)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.DrawImage($img, $dstRect, $srcRect, [System.Drawing.GraphicsUnit]::Pixel)
    $g.Dispose()
    $img.Dispose()
    return $bmp
}

function Get-Pixels {
    param([System.Drawing.Bitmap]$bmp)
    $rect = New-Object System.Drawing.Rectangle(0, 0, $bmp.Width, $bmp.Height)
    $bd = $bmp.LockBits($rect, [System.Drawing.Imaging.ImageLockMode]::ReadOnly, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $len = $bd.Stride * $bmp.Height
    $buf = New-Object byte[] $len
    [System.Runtime.InteropServices.Marshal]::Copy($bd.Scan0, $buf, 0, $len)
    $bmp.UnlockBits($bd)
    return ,$buf
}

function Build-BmpEntry {
    param([int]$size, [byte[]]$px)
    $hdr = New-Object byte[] 40
    [BitConverter]::GetBytes([uint32]40).CopyTo($hdr, 0)
    [BitConverter]::GetBytes([int32]$size).CopyTo($hdr, 4)
    [BitConverter]::GetBytes([int32]($size * 2)).CopyTo($hdr, 8)
    [BitConverter]::GetBytes([uint16]1).CopyTo($hdr, 12)
    [BitConverter]::GetBytes([uint16]32).CopyTo($hdr, 14)

    $rowLen = $size * 4
    $xorLen = $rowLen * $size
    $xor = New-Object byte[] $xorLen
    for ($y = 0; $y -lt $size; $y++) {
        $srcRow = $y * $rowLen
        $dstRow = ($size - 1 - $y) * $rowLen
        [Array]::Copy($px, $srcRow, $xor, $dstRow, $rowLen)
    }
    $maskRow = [int][Math]::Ceiling($size / 32.0) * 4
    $and = New-Object byte[] ($maskRow * $size)

    $imgData = New-Object byte[] (40 + $xorLen + $and.Length)
    [Array]::Copy($hdr, 0, $imgData, 0, 40)
    [Array]::Copy($xor, 0, $imgData, 40, $xorLen)
    [Array]::Copy($and, 0, $imgData, 40 + $xorLen, $and.Length)
    return ,$imgData
}

$entries = @()

foreach ($s in 16, 32) {
    $bmp = Make-Resized $s
    $px = Get-Pixels $bmp
    $blob = Build-BmpEntry $s $px
    $entries += ,@($s, $blob)
    $bmp.Dispose()
}

$bmp256 = Make-Resized 256
$ms = New-Object System.IO.MemoryStream
$bmp256.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
$pngBytes = $ms.ToArray()
$ms.Dispose()
$bmp256.Dispose()
$entries += ,@(0, $pngBytes)

$count = $entries.Count
$header = New-Object byte[] 6
[BitConverter]::GetBytes([uint16]0).CopyTo($header, 0)
[BitConverter]::GetBytes([uint16]1).CopyTo($header, 2)
[BitConverter]::GetBytes([uint16]$count).CopyTo($header, 4)

$dirLen = 16 * $count
$dir = New-Object byte[] $dirLen
$totalLen = 6 + $dirLen
for ($i = 0; $i -lt $count; $i++) {
    $totalLen += $entries[$i][1].Length
}
$out = New-Object byte[] $totalLen
[Array]::Copy($header, 0, $out, 0, 6)

$offset = 6 + $dirLen
$pos = 6 + $dirLen
for ($i = 0; $i -lt $count; $i++) {
    $e = $entries[$i]
    $base = $i * 16
    $dir[$base] = $e[0]
    $dir[$base + 1] = $e[0]
    $dir[$base + 2] = 0
    $dir[$base + 3] = 0
    [BitConverter]::GetBytes([uint16]1).CopyTo($dir, $base + 4)
    [BitConverter]::GetBytes([uint16]32).CopyTo($dir, $base + 6)
    [BitConverter]::GetBytes([uint32]$e[1].Length).CopyTo($dir, $base + 8)
    [BitConverter]::GetBytes([uint32]$offset).CopyTo($dir, $base + 12)
    [Array]::Copy($e[1], 0, $out, $pos, $e[1].Length)
    $pos += $e[1].Length
    $offset += $e[1].Length
}
[Array]::Copy($dir, 0, $out, 6, $dirLen)
[System.IO.File]::WriteAllBytes($outPath, $out)
Write-Host ("written " + $out.Length + " bytes, " + $count + " entries: 16 BMP, 32 BMP, 256 PNG")
