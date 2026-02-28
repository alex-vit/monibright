#ifndef AppVersion
  #define AppVersion "dev"
#endif

[Setup]
AppId={{F6592249-7FCA-4AE2-8D4D-A1CB6BE6836D}
AppName=MoniBright
AppVersion={#AppVersion}
AppVerName=MoniBright {#AppVersion}
AppPublisher=alex-vit
AppPublisherURL=https://github.com/alex-vit/monibright
DefaultDirName={localappdata}\MoniBright
DefaultGroupName=MoniBright
PrivilegesRequired=lowest
OutputDir=out
OutputBaseFilename=monibright-setup
SetupIconFile=icon\brightness.ico
UninstallDisplayIcon={app}\monibright.exe
Compression=lzma2
SolidCompression=yes
AppMutex=MoniBrightMutex
CloseApplications=yes
WizardStyle=modern

[Files]
Source: "out\monibright.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\MoniBright"; Filename: "{app}\monibright.exe"
Name: "{group}\Uninstall MoniBright"; Filename: "{uninstallexe}"

[Tasks]
Name: "autostart"; Description: "Start with Windows"; GroupDescription: "Additional options:"

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "MoniBright"; ValueData: """{app}\monibright.exe"""; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\monibright.exe"; Description: "Launch MoniBright"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{app}"

[UninstallRun]
Filename: "taskkill"; Parameters: "/F /IM monibright.exe"; Flags: runhidden; RunOnceId: "KillMoniBright"
