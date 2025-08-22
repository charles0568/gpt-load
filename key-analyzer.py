#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GPT-Load 密鑰分析工具
分析測試結果並生成詳細報告
"""

import pandas as pd
import json
import argparse
from datetime import datetime
import matplotlib.pyplot as plt
import seaborn as sns
from collections import Counter

class KeyAnalyzer:
    def __init__(self, csv_file: str):
        self.df = pd.read_csv(csv_file)
        self.total_keys = len(self.df)
        
    def basic_stats(self):
        """基本統計資訊"""
        valid_keys = self.df[self.df['valid'] == True]
        invalid_keys = self.df[self.df['valid'] == False]
        
        stats = {
            'total_keys': self.total_keys,
            'valid_keys': len(valid_keys),
            'invalid_keys': len(invalid_keys),
            'valid_percentage': len(valid_keys) / self.total_keys * 100,
            'avg_response_time': self.df[self.df['response_time_ms'] > 0]['response_time_ms'].mean(),
            'median_response_time': self.df[self.df['response_time_ms'] > 0]['response_time_ms'].median(),
        }
        
        return stats
    
    def error_analysis(self):
        """錯誤分析"""
        invalid_keys = self.df[self.df['valid'] == False]
        error_counts = Counter(invalid_keys['error_message'])
        
        return dict(error_counts.most_common())
    
    def group_analysis(self):
        """分組分析"""
        group_stats = self.df.groupby('group_id').agg({
            'valid': ['count', 'sum'],
            'response_time_ms': 'mean'
        }).round(2)
        
        group_stats.columns = ['total_keys', 'valid_keys', 'avg_response_time']
        group_stats['valid_percentage'] = (group_stats['valid_keys'] / group_stats['total_keys'] * 100).round(2)
        
        return group_stats
    
    def response_time_analysis(self):
        """回應時間分析"""
        valid_keys = self.df[(self.df['valid'] == True) & (self.df['response_time_ms'] > 0)]
        
        if len(valid_keys) == 0:
            return {}
        
        return {
            'min_response_time': valid_keys['response_time_ms'].min(),
            'max_response_time': valid_keys['response_time_ms'].max(),
            'avg_response_time': valid_keys['response_time_ms'].mean(),
            'median_response_time': valid_keys['response_time_ms'].median(),
            'p95_response_time': valid_keys['response_time_ms'].quantile(0.95),
            'p99_response_time': valid_keys['response_time_ms'].quantile(0.99),
        }
    
    def generate_report(self, output_file: str = None):
        """生成詳細報告"""
        if not output_file:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"key_analysis_report_{timestamp}.txt"
        
        stats = self.basic_stats()
        errors = self.error_analysis()
        groups = self.group_analysis()
        response_times = self.response_time_analysis()
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write("=" * 60 + "\n")
            f.write("GPT-Load 密鑰測試分析報告\n")
            f.write("=" * 60 + "\n")
            f.write(f"生成時間: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
            
            # 基本統計
            f.write("📊 基本統計\n")
            f.write("-" * 30 + "\n")
            f.write(f"總密鑰數量: {stats['total_keys']:,}\n")
            f.write(f"有效密鑰數量: {stats['valid_keys']:,}\n")
            f.write(f"無效密鑰數量: {stats['invalid_keys']:,}\n")
            f.write(f"有效率: {stats['valid_percentage']:.2f}%\n")
            f.write(f"平均回應時間: {stats['avg_response_time']:.2f} ms\n")
            f.write(f"中位數回應時間: {stats['median_response_time']:.2f} ms\n\n")
            
            # 錯誤分析
            f.write("❌ 錯誤分析\n")
            f.write("-" * 30 + "\n")
            for error, count in errors.items():
                percentage = count / stats['invalid_keys'] * 100 if stats['invalid_keys'] > 0 else 0
                f.write(f"{error}: {count} ({percentage:.1f}%)\n")
            f.write("\n")
            
            # 分組分析
            f.write("📁 分組分析\n")
            f.write("-" * 30 + "\n")
            f.write(groups.to_string())
            f.write("\n\n")
            
            # 回應時間分析
            if response_times:
                f.write("⏱️ 回應時間分析 (僅有效密鑰)\n")
                f.write("-" * 30 + "\n")
                f.write(f"最小回應時間: {response_times['min_response_time']:.2f} ms\n")
                f.write(f"最大回應時間: {response_times['max_response_time']:.2f} ms\n")
                f.write(f"平均回應時間: {response_times['avg_response_time']:.2f} ms\n")
                f.write(f"中位數回應時間: {response_times['median_response_time']:.2f} ms\n")
                f.write(f"95% 分位數: {response_times['p95_response_time']:.2f} ms\n")
                f.write(f"99% 分位數: {response_times['p99_response_time']:.2f} ms\n\n")
        
        print(f"詳細報告已儲存到: {output_file}")
        return output_file
    
    def export_valid_keys(self, output_file: str = None):
        """匯出有效密鑰清單"""
        if not output_file:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"valid_keys_{timestamp}.csv"
        
        valid_keys = self.df[self.df['valid'] == True]
        valid_keys.to_csv(output_file, index=False, encoding='utf-8')
        
        print(f"有效密鑰清單已匯出到: {output_file}")
        return output_file
    
    def export_invalid_keys(self, output_file: str = None):
        """匯出無效密鑰清單"""
        if not output_file:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"invalid_keys_{timestamp}.csv"
        
        invalid_keys = self.df[self.df['valid'] == False]
        invalid_keys.to_csv(output_file, index=False, encoding='utf-8')
        
        print(f"無效密鑰清單已匯出到: {output_file}")
        return output_file

def main():
    parser = argparse.ArgumentParser(description='GPT-Load 密鑰分析工具')
    parser.add_argument('csv_file', help='測試結果 CSV 檔案')
    parser.add_argument('--report', action='store_true', help='生成詳細報告')
    parser.add_argument('--valid-keys', action='store_true', help='匯出有效密鑰')
    parser.add_argument('--invalid-keys', action='store_true', help='匯出無效密鑰')
    
    args = parser.parse_args()
    
    try:
        analyzer = KeyAnalyzer(args.csv_file)
        
        # 顯示基本統計
        stats = analyzer.basic_stats()
        print("=" * 50)
        print("基本統計資訊")
        print("=" * 50)
        print(f"總密鑰數量: {stats['total_keys']:,}")
        print(f"有效密鑰數量: {stats['valid_keys']:,}")
        print(f"無效密鑰數量: {stats['invalid_keys']:,}")
        print(f"有效率: {stats['valid_percentage']:.2f}%")
        print(f"平均回應時間: {stats['avg_response_time']:.2f} ms")
        
        if args.report:
            analyzer.generate_report()
        
        if args.valid_keys:
            analyzer.export_valid_keys()
        
        if args.invalid_keys:
            analyzer.export_invalid_keys()
            
    except Exception as e:
        print(f"錯誤: {e}")

if __name__ == "__main__":
    main()
