@echo off
REM GPT-Load 批量密鑰測試工具 - Windows 批次檔案
REM 使用方法: test-keys.bat

echo ========================================
echo GPT-Load 批量密鑰測試工具
echo ========================================
echo.

REM 設定參數
set GPT_LOAD_URL=http://192.168.1.99:3001
set AUTH_KEY=sk-123456
set CONCURRENT=100

echo 配置資訊:
echo - GPT-Load 地址: %GPT_LOAD_URL%
echo - 認證密鑰: %AUTH_KEY%
echo - 併發數: %CONCURRENT%
echo.

REM 檢查 Python 是否安裝
python --version >nul 2>&1
if errorlevel 1 (
    echo 錯誤: 未找到 Python，請先安裝 Python 3.7+
    echo 下載地址: https://www.python.org/downloads/
    pause
    exit /b 1
)

REM 安裝必要的 Python 套件
echo 正在安裝必要的 Python 套件...
pip install aiohttp asyncio

REM 執行測試
echo.
echo 開始批量測試密鑰...
echo 注意: 2萬個密鑰預計需要 10-30 分鐘完成
echo.

python batch-key-tester.py --url %GPT_LOAD_URL% --auth-key %AUTH_KEY% --concurrent %CONCURRENT%

echo.
echo 測試完成！結果已儲存到 CSV 檔案。
echo.
pause
