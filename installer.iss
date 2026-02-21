#ifndef AppVersion
  #define AppVersion "dev"
#endif

[Setup]
AppName=MoniBright
AppVersion={#AppVersion}
AppPublisher=alex-vit
AppPublisherURL=https://github.com/alex-vit/monibright
DefaultDirName={localappdata}\MoniBright
DefaultGroupName=MoniBright
PrivilegesRequired=lowest
OutputBaseFilename=monibright-setup
SetupIconFile=icon\brightness.ico
UninstallDisplayIcon={app}\monibright.exe
Compression=lzma2
SolidCompression=yes
AppMutex=MoniBrightMutex
CloseApplications=yes

[Files]
Source: "monibright.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\MoniBright"; Filename: "{app}\monibright.exe"
Name: "{group}\Uninstall MoniBright"; Filename: "{uninstallexe}"

[Tasks]
Name: "autostart"; Description: "Start with Windows"; GroupDescription: "Additional options:"

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "MoniBright"; ValueData: "{app}\monibright.exe"; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\monibright.exe"; Description: "Launch MoniBright"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill"; Parameters: "/F /IM monibright.exe"; Flags: runhidden; RunOnceId: "KillMoniBright"
