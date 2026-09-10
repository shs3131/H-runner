; Hrunner NSIS Installer Script
; Packages Hrunner Main App into dist\HrunnerSetup.exe

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "WinMessages.nsh"
!include "LogicLib.nsh"

Name "Hrunner"
OutFile "..\..\dist\HrunnerSetup.exe"
InstallDir "$LOCALAPPDATA\Programs\Hrunner"
InstallDirRegKey HKCU "Software\Hrunner" "InstallPath"
RequestExecutionLevel user

; Modern UI Configuration
!define MUI_ABORTWARNING

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\hmanager.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Open Hrunner Manager"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Hrunner Core" SecCore
    SetOutPath "$INSTDIR"

    ; Core binaries
    File "..\..\hrunner.exe"
    File "..\..\hmanager.exe"
    File "..\..\hlauncher.exe"
    File "..\..\hbuild.exe"

    ; Write registry entries for application discovery
    WriteRegStr HKCU "Software\Hrunner" "InstallPath" "$INSTDIR"
    WriteRegStr HKCU "Software\Hrunner" "Version" "1.0.1"

    ; App Paths registration so hrunner/hmanager can be launched via Win+R or shell
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hrunner.exe" "" "$INSTDIR\hrunner.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hrunner.exe" "Path" "$INSTDIR"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hmanager.exe" "" "$INSTDIR\hmanager.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hmanager.exe" "Path" "$INSTDIR"

    ; Add to User PATH via registry
    ReadRegStr $0 HKCU "Environment" "PATH"
    ${If} $0 == ""
        WriteRegExpandStr HKCU "Environment" "PATH" "$INSTDIR"
    ${Else}
        ; Check if already in PATH
        Push "$0"
        Push "$INSTDIR"
        WriteRegExpandStr HKCU "Environment" "PATH" "$0;$INSTDIR"
    ${EndIf}
    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=2000

    ; Write Uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Add/Remove Programs entry
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner" "DisplayName" "Hrunner Runtime & Package Manager"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner" "UninstallString" '"$INSTDIR\Uninstall.exe"'
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner" "DisplayIcon" "$INSTDIR\hrunner.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner" "DisplayVersion" "1.0.1"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner" "Publisher" "Hrunner Project"

    ; Start Menu Shortcut
    CreateDirectory "$SMPROGRAMS\Hrunner"
    CreateShortcut "$SMPROGRAMS\Hrunner\Hrunner Manager.lnk" "$INSTDIR\hmanager.exe"
    CreateShortcut "$SMPROGRAMS\Hrunner\Uninstall Hrunner.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Uninstall"
    ; Remove shortcuts
    Delete "$SMPROGRAMS\Hrunner\Hrunner Manager.lnk"
    Delete "$SMPROGRAMS\Hrunner\Uninstall Hrunner.lnk"
    RMDir "$SMPROGRAMS\Hrunner"

    ; Remove files
    Delete "$INSTDIR\hrunner.exe"
    Delete "$INSTDIR\hmanager.exe"
    Delete "$INSTDIR\hlauncher.exe"
    Delete "$INSTDIR\hbuild.exe"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"

    ; Remove registry keys
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Hrunner"
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hrunner.exe"
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\App Paths\hmanager.exe"
    DeleteRegKey HKCU "Software\Hrunner"

    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=2000
SectionEnd
