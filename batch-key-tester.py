#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GPT-Load 批量密鑰測試工具
支援大規模 API 密鑰有效性檢測
"""

import asyncio
import aiohttp
import json
import csv
import time
from datetime import datetime
from typing import List, Dict, Tuple
import argparse
import logging

# 設定日誌
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('key_test_results.log', encoding='utf-8'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

class KeyTester:
    def __init__(self, gpt_load_url: str, auth_key: str, max_concurrent: int = 50):
        self.gpt_load_url = gpt_load_url.rstrip('/')
        self.auth_key = auth_key
        self.max_concurrent = max_concurrent
        self.semaphore = asyncio.Semaphore(max_concurrent)
        self.results = []
        
    async def test_single_key(self, session: aiohttp.ClientSession, group_id: int, key_id: int, api_key: str) -> Dict:
        """測試單個 API 密鑰"""
        async with self.semaphore:
            try:
                # 使用 GPT-Load 的密鑰測試 API
                url = f"{self.gpt_load_url}/api/keys/{key_id}/test"
                headers = {
                    'Authorization': f'Bearer {self.auth_key}',
                    'Content-Type': 'application/json'
                }
                
                start_time = time.time()
                async with session.post(url, headers=headers, timeout=30) as response:
                    end_time = time.time()
                    response_time = round((end_time - start_time) * 1000, 2)  # ms
                    
                    result = {
                        'key_id': key_id,
                        'api_key': api_key[:20] + '...' if len(api_key) > 20 else api_key,
                        'group_id': group_id,
                        'status_code': response.status,
                        'response_time_ms': response_time,
                        'timestamp': datetime.now().isoformat(),
                        'valid': response.status == 200,
                        'error_message': ''
                    }
                    
                    if response.status != 200:
                        try:
                            error_data = await response.json()
                            result['error_message'] = error_data.get('message', f'HTTP {response.status}')
                        except:
                            result['error_message'] = f'HTTP {response.status}'
                    
                    return result
                    
            except asyncio.TimeoutError:
                return {
                    'key_id': key_id,
                    'api_key': api_key[:20] + '...' if len(api_key) > 20 else api_key,
                    'group_id': group_id,
                    'status_code': 0,
                    'response_time_ms': 30000,
                    'timestamp': datetime.now().isoformat(),
                    'valid': False,
                    'error_message': 'Timeout'
                }
            except Exception as e:
                return {
                    'key_id': key_id,
                    'api_key': api_key[:20] + '...' if len(api_key) > 20 else api_key,
                    'group_id': group_id,
                    'status_code': 0,
                    'response_time_ms': 0,
                    'timestamp': datetime.now().isoformat(),
                    'valid': False,
                    'error_message': str(e)
                }

    async def get_all_keys(self, session: aiohttp.ClientSession) -> List[Dict]:
        """獲取所有 API 密鑰"""
        try:
            url = f"{self.gpt_load_url}/api/keys"
            headers = {'Authorization': f'Bearer {self.auth_key}'}
            
            async with session.get(url, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    return data.get('data', [])
                else:
                    logger.error(f"獲取密鑰列表失敗: HTTP {response.status}")
                    return []
        except Exception as e:
            logger.error(f"獲取密鑰列表異常: {e}")
            return []

    async def batch_test_keys(self, keys: List[Dict] = None) -> List[Dict]:
        """批量測試密鑰"""
        connector = aiohttp.TCPConnector(limit=self.max_concurrent * 2)
        timeout = aiohttp.ClientTimeout(total=60)
        
        async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
            if keys is None:
                logger.info("正在獲取所有密鑰...")
                keys = await self.get_all_keys(session)
            
            if not keys:
                logger.error("沒有找到任何密鑰")
                return []
            
            logger.info(f"開始測試 {len(keys)} 個密鑰，併發數: {self.max_concurrent}")
            
            # 建立測試任務
            tasks = []
            for key_info in keys:
                task = self.test_single_key(
                    session,
                    key_info.get('group_id', 0),
                    key_info.get('id', 0),
                    key_info.get('key', '')
                )
                tasks.append(task)
            
            # 執行測試並顯示進度
            completed = 0
            results = []
            
            for coro in asyncio.as_completed(tasks):
                result = await coro
                results.append(result)
                completed += 1
                
                if completed % 100 == 0 or completed == len(tasks):
                    valid_count = sum(1 for r in results if r['valid'])
                    logger.info(f"進度: {completed}/{len(tasks)} ({completed/len(tasks)*100:.1f}%) - 有效: {valid_count}")
            
            return results

    def save_results(self, results: List[Dict], filename: str = None):
        """儲存測試結果"""
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"key_test_results_{timestamp}.csv"
        
        with open(filename, 'w', newline='', encoding='utf-8') as csvfile:
            if results:
                fieldnames = results[0].keys()
                writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
                writer.writeheader()
                writer.writerows(results)
        
        logger.info(f"結果已儲存到: {filename}")

    def print_summary(self, results: List[Dict]):
        """列印測試摘要"""
        if not results:
            logger.info("沒有測試結果")
            return
        
        total = len(results)
        valid = sum(1 for r in results if r['valid'])
        invalid = total - valid
        
        avg_response_time = sum(r['response_time_ms'] for r in results if r['response_time_ms'] > 0) / max(1, len([r for r in results if r['response_time_ms'] > 0]))
        
        logger.info("=" * 50)
        logger.info("測試結果摘要")
        logger.info("=" * 50)
        logger.info(f"總密鑰數: {total}")
        logger.info(f"有效密鑰: {valid} ({valid/total*100:.1f}%)")
        logger.info(f"無效密鑰: {invalid} ({invalid/total*100:.1f}%)")
        logger.info(f"平均回應時間: {avg_response_time:.2f} ms")
        logger.info("=" * 50)

async def main():
    parser = argparse.ArgumentParser(description='GPT-Load 批量密鑰測試工具')
    parser.add_argument('--url', required=True, help='GPT-Load 服務地址 (例如: http://192.168.1.99:3001)')
    parser.add_argument('--auth-key', required=True, help='GPT-Load 認證密鑰')
    parser.add_argument('--concurrent', type=int, default=50, help='併發數 (預設: 50)')
    parser.add_argument('--output', help='輸出檔案名稱')
    
    args = parser.parse_args()
    
    tester = KeyTester(args.url, args.auth_key, args.concurrent)
    
    start_time = time.time()
    results = await tester.batch_test_keys()
    end_time = time.time()
    
    if results:
        tester.save_results(results, args.output)
        tester.print_summary(results)
        logger.info(f"總耗時: {end_time - start_time:.2f} 秒")
    else:
        logger.error("測試失敗，沒有獲得結果")

if __name__ == "__main__":
    asyncio.run(main())
