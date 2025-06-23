$toolsDir    = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

$packageArgs = @{
  packageName    = 'foks'
  url            = 'https://github.com/foks-proj/go-foks/releases/download/v0.0.21/foks-v0.0.21-win-choco-x86.zip'
  url64          = 'https://github.com/foks-proj/go-foks/releases/download/v0.0.21/foks-v0.0.21-win-choco-amd64.zip'
  checksum64     = 'dcd28d09396c63ebcee05273abe6fbf75d7c9e1b7a55bebde7fbee2a7d9f47bf'
  checksum       = 'ddca451bb1f49f0238d110f48f4d87f420dcf375db08275eb2793095662015db'
  checksumType   = 'sha256'
  checksumType64 = 'sha256'
  unzipLocation  = $toolsDir
}

Install-ChocolateyZipPackage @packageArgs

# Need to copy the item over since we need to know the equivalent of os.Args[0]
# inside the executable, and we lose that via the shimming process.
Copy-Item "$toolsDir\foks.exe" "$toolsDir\git-remote-foks.exe" -Force

Install-BinFile `
  -Name 'foks' `
  -Path "$toolsDir\foks.exe"

Install-BinFile `
  -Name 'git-remote-foks' `
  -Path "$toolsDir\git-remote-foks.exe"
