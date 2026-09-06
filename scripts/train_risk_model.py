# scripts/train_risk_model.py
import pandas as pd
import numpy as np
from sklearn.ensemble import IsolationForest
import joblib
from datetime import datetime, timedelta
import random

# 生成模拟用户历史数据（实际应使用真实日志）
def generate_mock_history(num_users=1000, days=30):
    users = []
    for uid in range(num_users):
        history = []
        num_requests = random.randint(1, 200)
        for _ in range(num_requests):
            timestamp = datetime.now() - timedelta(
                hours=random.randint(0, 24*30),
                minutes=random.randint(0, 60)
            )
            # 随机攻击标记，正常用户约5%攻击率，异常用户可能>30%
            is_attack = random.random() < (0.3 if uid % 10 == 0 else 0.05)
            # 模拟IP和请求类型
            ip = f"192.168.{random.randint(0,255)}.{random.randint(0,255)}"
            req_type = random.choice(["chat", "query", "command", "file"])
            history.append({
                "timestamp": timestamp.isoformat(),
                "is_attack": is_attack,
                "ip": ip,
                "type": req_type,
            })
        users.append({"user_id": str(uid), "history": history})
    return users

# 从历史中提取特征（与 risk_scorer.py 保持一致）
def extract_features_from_history(history):
    if not history:
        return np.zeros(7)
    df = pd.DataFrame(history)
    df["timestamp"] = pd.to_datetime(df["timestamp"])
    now = datetime.now()
    df_1h = df[df["timestamp"] > now - pd.Timedelta(hours=1)]
    df_24h = df[df["timestamp"] > now - pd.Timedelta(hours=24)]
    call_count_1h = len(df_1h)
    call_count_24h = len(df_24h)
    night_mask = (df["timestamp"].dt.hour >= 22) | (df["timestamp"].dt.hour < 6)
    night_rate = night_mask.sum() / len(df) if len(df) > 0 else 0
    if len(df) > 1:
        sorted_df = df.sort_values("timestamp")
        intervals = sorted_df["timestamp"].diff().dt.total_seconds().dropna()
        avg_interval = intervals.mean()
    else:
        avg_interval = 0
    attack_rate = df["is_attack"].sum() / len(df) if "is_attack" in df else 0
    # 新增：unique_ip_count
    unique_ip_count = df["ip"].nunique() if "ip" in df else 1
    # 新增：请求类型熵
    if "type" in df:
        type_counts = df["type"].value_counts(normalize=True)
        entropy = -sum(p * np.log2(p) for p in type_counts if p > 0)
    else:
        entropy = 0.5
    features = np.array([
        call_count_1h,
        call_count_24h,
        night_rate,
        min(avg_interval, 3600),
        unique_ip_count,
        attack_rate,
        entropy
    ])
    return features

if __name__ == "__main__":
    # 生成训练数据
    mock_users = generate_mock_history(2000)  # 2000个用户
    X = []
    for user in mock_users:
        feat = extract_features_from_history(user["history"])
        X.append(feat)
    X = np.vstack(X)

    # 训练隔离森林
    model = IsolationForest(
        n_estimators=200,
        contamination=0.05,
        random_state=42
    )
    model.fit(X)

    # 保存模型
    os.makedirs("./models", exist_ok=True)
    joblib.dump(model, "./models/risk_model.joblib")
    print("风险模型已保存至 ./models/risk_model.joblib")