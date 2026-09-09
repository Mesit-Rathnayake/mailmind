$WshShell = New-Object -ComObject WScript.Shell
$DesktopPath = [System.Environment]::GetFolderPath([System.Environment+SpecialFolder]::Desktop)
$ShortcutPath = Join-Path $DesktopPath "MailMind.lnk"
$TargetPath = Join-Path (Get-Location) "start.bat"
$WorkingDir = (Get-Location).Path

# Locate icon using environment variable
$SourceIcon = Join-Path $env:USERPROFILE "Downloads\message.ico"
$LocalIcon = Join-Path (Get-Location) "icon.ico"

if (Test-Path $SourceIcon) {
    Copy-Item -Path $SourceIcon -Destination $LocalIcon -Force
}

# Create / Update Desktop Shortcut with icon
$Shortcut = $WshShell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = $TargetPath
$Shortcut.WorkingDirectory = $WorkingDir
$Shortcut.Description = "Launch MailMind AI Email Client"
if (Test-Path $LocalIcon) {
    $Shortcut.IconLocation = "$LocalIcon,0"
}
$Shortcut.Save()

# Also create / update MailMind.lnk directly in the project folder
$ProjectShortcutPath = Join-Path (Get-Location) "MailMind.lnk"
$ProjectShortcut = $WshShell.CreateShortcut($ProjectShortcutPath)
$ProjectShortcut.TargetPath = $TargetPath
$ProjectShortcut.WorkingDirectory = $WorkingDir
$ProjectShortcut.Description = "Launch MailMind AI Email Client"
if (Test-Path $LocalIcon) {
    $ProjectShortcut.IconLocation = "$LocalIcon,0"
}
$ProjectShortcut.Save()

Write-Host "MailMind desktop shortcut created with custom icon at: $ShortcutPath"
Write-Host "MailMind project shortcut created with custom icon at: $ProjectShortcutPath"
