' 1Panel.edu monitor-daemon hidden launcher (Windows)
' 用途: 开机后静默启动 monitor-daemon, 不闪 cmd 黑窗
' 部署: 复制到 %APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\1panel-monitor.vbs
Set WshShell = CreateObject("WScript.Shell")
WshShell.Run chr(34) & "C:\Python312\python.exe" & chr(34) & " " & chr(34) & "D:\MiniMax Code\1Panel-edu-research\.scheduler\monitor-daemon.py" & chr(34), 0, False
Set WshShell = Nothing
