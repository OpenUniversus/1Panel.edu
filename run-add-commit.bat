@echo off
cd /d "D:\MiniMax Code\1Panel-edu-research"
git add -A
git -c user.email=bot@1panel-edu.local -c user.name=1PanelEduBot commit -m "%~1"
git push origin main 2>nul
