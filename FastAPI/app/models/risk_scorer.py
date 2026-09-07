# risk_scorer.py
import numpy as np
import pandas as pd
from sklearn.ensemble import IsolationForest
from datetime import datetime
import joblib
from typing import List, Dict

class RiskScorer:
    def __init__(self, model_path: str = None):
        if model_path:
            self.model = joblib.load(model_path)
        else:
            # 初始化新模型
            self.model = IsolationForest(
                n_estimators=200,
                contamination=0.05,  # 预期5%的异常率[reference:13]
                random_state=42
            )
        self.feature_names = [
            "call_count_1h",      # 1小时调用次数
            "call_count_24h",     # 24小时调用次数
            "night_rate",         # 夜间(22-6点)调用占比
            "avg_interval",       # 平均调用间隔(秒)
            "unique_ip_count",    # 不同IP数量
            "attack_rate",        # 触发检测的比例
            "request_type_entropy" # 请求类型熵
        ]
    
    def extract_features(self, user_history: List[Dict]) -> np.ndarray:
        """
        从用户历史记录中提取特征
        user_history: [{"timestamp": "2026-08-16 10:00:00", "type": "chat", "is_attack": False, "ip": "1.2.3.4"}, ...]
        """
        if not user_history:
            return np.zeros(len(self.feature_names))
        
        df = pd.DataFrame(user_history)
        df["timestamp"] = pd.to_datetime(df["timestamp"])
        now = datetime.now()
        
        # 1小时 & 24小时调用次数
        df_1h = df[df["timestamp"] > now - pd.Timedelta(hours=1)]
        df_24h = df[df["timestamp"] > now - pd.Timedelta(hours=24)]
        
        call_count_1h = len(df_1h)
        call_count_24h = len(df_24h)
        
        # 夜间调用占比
        night_mask = (df["timestamp"].dt.hour >= 22) | (df["timestamp"].dt.hour < 6)
        night_rate = night_mask.sum() / len(df) if len(df) > 0 else 0
        
        # 平均调用间隔（秒）
        if len(df) > 1:
            sorted_df = df.sort_values("timestamp")
            intervals = sorted_df["timestamp"].diff().dt.total_seconds().dropna()
            avg_interval = intervals.mean()
        else:
            avg_interval = 0
        
        # 攻击触发比例
        if "is_attack" in df.columns:
            attack_rate = df["is_attack"].sum() / len(df)
        else:
            attack_rate = 0
        
        # 不同 IP 数量（若存在 ip 列）
        if "ip" in df.columns:
            unique_ip_count = df["ip"].nunique()
        else:
            unique_ip_count = 1  # 默认值

        # 请求类型熵（若存在 type 列）
        if "type" in df.columns:
            type_counts = df["type"].value_counts(normalize=True)
            entropy = -sum(p * np.log2(p) for p in type_counts if p > 0)
        else:
            entropy = 0.5  # 默认值

        features = np.array([
            call_count_1h,
            call_count_24h,
            night_rate,
            min(avg_interval, 3600),  # 上限1小时
            unique_ip_count,
            attack_rate,
            entropy
        ])
        return features.reshape(1, -1)
    
    def predict(self, user_history: List[Dict]) -> dict:
        """
        计算用户风险评分 0-100
        返回值: {"score": 0-100, "is_anomaly": bool, "level": "low|medium|high"}
        """
        features = self.extract_features(user_history)
        
        # 孤立森林预测: 1=正常, -1=异常
        pred = self.model.predict(features)[0]
        
        # 计算异常分数 (0-1, 越高越异常)
        # 使用decision_function获取置信度
        anomaly_score = -self.model.score_samples(features)[0]
        # 归一化到0-100
        risk_score = min(100, max(0, anomaly_score * 50))
        
        if pred == -1:
            risk_score = max(risk_score, 60)  # 异常至少60分
        
        # 分级
        if risk_score < 30:
            level = "low"
        elif risk_score < 60:
            level = "medium"
        else:
            level = "high"
        
        return {
            "score": round(risk_score, 1),
            "is_anomaly": pred == -1,
            "level": level,
            "features": dict(zip(self.feature_names, features[0].tolist()))
        }
    
    def train(self, historical_data: List[Dict]):
        """离线训练/更新模型"""
        features = []
        for user in historical_data:
            feat = self.extract_features(user["history"])
            features.append(feat)
        X = np.vstack(features)
        self.model.fit(X)
        joblib.dump(self.model, "risk_model.joblib")
