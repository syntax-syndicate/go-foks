$url64       = "https://github.com/foks-proj/go-foks/releases/download/v0.0.20/foks-v0.0.20-win-choco-amd64.zip"
$url         = "https://github.com/foks-proj/go-foks/releases/download/v0.0.20/foks-v0.0.20-win-choco-x86.zip"
$checksum    = "93a76d3dcbc6d47ba0cdaa5a32884d58159af8d04caec9b1fd0307e40507526a"
$checksum64  = "6d9a663c2582d4a326fa513ae5ebbb2f03d8b1007be94c4e35771faa8bc0f097"
$packageName = "foks"
$toolsDir    = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

Install-ChocolateyZipPackage `
  -PackageName   $packageName `
  -FileType      'zip' `
  -Url            $url `
  -Url64bit       $url64 `
  -Checksum       $checksum `
  -ChecksumType   'sha256' `
  -Checksum64     $checksum64 `
  -ChecksumType64 'sha256' `
  -UnzipLocation  $toolsDir

# Need to copy the item over since we need to know the equivalent of os.Args[0]
# inside the executable, and we lose that via the shimming process.
Copy-Item "$toolsDir\foks.exe" "$toolsDir\git-remote-foks.exe" -Force

Install-BinFile `
  -Name 'foks' `
  -Path "$toolsDir\foks.exe"

Install-BinFile `
  -Name 'git-remote-foks' `
  -Path "$toolsDir\git-remote-foks.exe"
