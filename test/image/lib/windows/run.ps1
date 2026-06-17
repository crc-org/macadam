param(
    [Parameter(HelpMessage='pass the path of the image')]
    $imageLocation="",
    [Parameter(HelpMessage='The image url to download the image from')]
    $imageUrl="",
    [Parameter(HelpMessage='Name of the folder on the target host under $HOME where all the content will be copied')]
    $targetFolder="macadam-test",
    [Parameter(HelpMessage='the path of the macadam binary to be tested')]
    $macadamBinary="",
    [Parameter(HelpMessage='ginkgo label filter to run specific tests')]
    $ginkgoLabelFilter=""
)


Write-Host "Updating WSL kernel ..."
wsl --update
wsl --shutdown

Move-Item -Path "$targetFolder\bin\e2e.test" -Destination "$targetFolder\bin\e2e.test.exe" -Force
Move-Item -Path "$targetFolder\bin\macadam" -Destination "$targetFolder\bin\macadam.exe" -Force

if ($macadamBinary -ne "") {
    Copy-Item -Path $macadamBinary -Destination "$targetFolder\bin\macadam.exe" -Force
}
$macadamBinary = "$HOME\$targetFolder\bin\macadam.exe"

if ($imageLocation -eq "" -and $imageUrl -ne "") {
    
    $filename = [System.IO.Path]::GetFileName($imageUrl)
    Write-Host "Download image $filename  ..."
    $downloadImage = "$HOME\$targetFolder\testdata\$filename"
    Invoke-WebRequest -Uri $imageUrl -OutFile $downloadImage
    if ($filename.ToLower().EndsWith(".vhd") -or $filename.ToLower().EndsWith(".vhdx") -or $filename.ToLower().EndsWith(".wsl") -or $filename.ToLower().EndsWith(".tar.gz")) {
        $imageLocation = $downloadImage
    } else {
        Write-Host "not supported format $filename"
    }
}

if ($ginkgoLabelFilter -eq "") {
    $ginkgoLabelFilter="windows"
} else {
    $ginkgoLabelFilter="windows&&$ginkgoLabelFilter"
}


Set-Location "$targetFolder\bin"
Get-ChildItem
.\e2e.test.exe --macadam-binary="$macadamBinary" --ginkgo.label-filter="$ginkgoLabelFilter" --image=$imageLocation *>&1 | Tee-Object -FilePath "e2e.result"

Set-Location ..
New-Item -ItemType Directory -Path "results" -Force | Out-Null
Move-Item -Path "bin\out\*" -Destination "results\" -Force
Move-Item -Path "bin\e2e.result" -Destination "results\" -Force
