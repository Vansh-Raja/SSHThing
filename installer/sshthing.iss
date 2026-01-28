#define AppName "SSHThing"
#define AppExeName "sshthing.exe"
#define AppPublisher "Vansh Raja"
#define AppURL "https://github.com/Vansh-Raja/SSHThing"

; These can be overridden by the build (e.g. CI) via iscc /D...
#ifndef MyAppVersion
  #define MyAppVersion "0.0.0"
#endif

#ifndef MyAppVersionInfoVersion
  #define MyAppVersionInfoVersion MyAppVersion + ".0"
#endif

[Setup]
AppId={{E3E3521E-2E7D-4B3A-9E2E-0E6F67D6B3B9}
AppName={#AppName}
AppVersion={#MyAppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}

ChangesEnvironment=yes

DefaultDirName={localappdata}\Programs\{#AppName}
DefaultGroupName={#AppName}
DisableDirPage=no
DisableProgramGroupPage=no

OutputDir=..
OutputBaseFilename=sshthing-setup-windows-amd64

Compression=lzma2
SolidCompression=yes

PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

VersionInfoVersion={#MyAppVersionInfoVersion}
VersionInfoCompany={#AppPublisher}
VersionInfoDescription={#AppName}
VersionInfoProductName={#AppName}
VersionInfoProductVersion={#MyAppVersion}
VersionInfoTextVersion={#MyAppVersion}

WizardStyle=modern
LicenseFile=..\LICENSE

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop icon"; GroupDescription: "Additional icons:"; Flags: unchecked
Name: "addtopath"; Description: "Add {#AppName} to PATH (current user)"; GroupDescription: "Additional tasks:"

[Files]
Source: "..\sshthing.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Launch {#AppName}"; Flags: nowait postinstall skipifsilent

[Code]
function NeedsAddPath(const AppDir: string): Boolean;
var
  OrigPath: string;
  UOrigPath: string;
  UAppDir: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
    OrigPath := '';

  UOrigPath := ';' + Uppercase(OrigPath) + ';';
  UAppDir := ';' + Uppercase(AppDir) + ';';
  Result := Pos(UAppDir, UOrigPath) = 0;
end;

procedure AddToPath(const AppDir: string);
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
    OrigPath := '';

  if OrigPath = '' then
    RegWriteExpandStringValue(HKCU, 'Environment', 'Path', AppDir)
  else
    RegWriteExpandStringValue(HKCU, 'Environment', 'Path', OrigPath + ';' + AppDir);
end;

procedure RemoveFromPath(const AppDir: string);
var
  OrigPath: string;
  NewPath: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
    Exit;

  NewPath := OrigPath;

  StringChangeEx(NewPath, ';' + AppDir + ';', ';', True);
  StringChangeEx(NewPath, ';' + AppDir, '', True);
  StringChangeEx(NewPath, AppDir + ';', '', True);
  if CompareText(NewPath, AppDir) = 0 then
    NewPath := '';

  while Pos(';;', NewPath) > 0 do
    StringChangeEx(NewPath, ';;', ';', True);
  if (Length(NewPath) > 0) and (NewPath[1] = ';') then
    Delete(NewPath, 1, 1);
  if (Length(NewPath) > 0) and (NewPath[Length(NewPath)] = ';') then
    Delete(NewPath, Length(NewPath), 1);

  RegWriteExpandStringValue(HKCU, 'Environment', 'Path', NewPath);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    if WizardIsTaskSelected('addtopath') then
    begin
      if NeedsAddPath(ExpandConstant('{app}')) then
        AddToPath(ExpandConstant('{app}'));
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
    RemoveFromPath(ExpandConstant('{app}'));
end;
