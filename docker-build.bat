@echo off
REM GPT-Load 繁體中文版 Docker 建置腳本 (Windows)
REM 使用方法: docker-build.bat

echo 🐳 開始建置 GPT-Load 繁體中文版 Docker 映像...

REM 設定變數
set DOCKER_USERNAME=charles0568
set DOCKER_PASSWORD=T8WjC9n94tmTt9
set IMAGE_NAME=charles0568/gpt-load
set VERSION=v1.0.0-zh-tw

REM 登入 DockerHub
echo 🔐 登入 DockerHub...
echo %DOCKER_PASSWORD% | docker login -u %DOCKER_USERNAME% --password-stdin

REM 建置前端
echo 🏗️ 建置前端...
cd web
call npm install
call npm run build
cd ..

REM 建置 Docker 映像
echo 🐳 建置 Docker 映像...
docker build -t %IMAGE_NAME%:%VERSION% .
docker build -t %IMAGE_NAME%:zh-tw .
docker build -t %IMAGE_NAME%:latest-zh-tw .

REM 推送到 DockerHub
echo 📤 推送到 DockerHub...
docker push %IMAGE_NAME%:%VERSION%
docker push %IMAGE_NAME%:zh-tw
docker push %IMAGE_NAME%:latest-zh-tw

echo ✅ Docker 映像建置和推送完成！
echo.
echo 📋 可用的映像標籤：
echo   - %IMAGE_NAME%:%VERSION%
echo   - %IMAGE_NAME%:zh-tw
echo   - %IMAGE_NAME%:latest-zh-tw
echo.
echo 🚀 使用方式：
echo   docker run -d --name gpt-load-zh-tw ^
echo     -p 3001:3001 ^
echo     -e AUTH_KEY=sk-123456 ^
echo     -v "%cd%/data":/app/data ^
echo     %IMAGE_NAME%:zh-tw

pause
