# main.py
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import uvicorn

from app.models.prompt_detector import PromptDetector
from app.models.desensitizer import Desensitizer
from app.models.risk_scorer import RiskScorer

app = FastAPI(title="AI安全风控服务", version="1.0.0")

# 初始化模型（启动时加载）
detector = PromptDetector("Qwen/Qwen2-1.5B-Instruct", "./adapters/prompt_safety")
desensitizer = Desensitizer()
risk_scorer = RiskScorer("./models/risk_model.joblib")

# ============ 请求/响应模型 ============
class PromptDetectRequest(BaseModel):
    text: str

class PromptDetectResponse(BaseModel):
    is_attack: bool
    confidence: float
    reason: str
    layer: str

class DesensitizeRequest(BaseModel):
    text: str

class DesensitizeResponse(BaseModel):
    desensitized_text: str
    detected_entities: List[dict]
    compliance: dict

class RiskScoreRequest(BaseModel):
    user_id: str
    history: List[dict]

class RiskScoreResponse(BaseModel):
    user_id: str
    score: float
    is_anomaly: bool
    level: str
    features: dict

# ============ API 端点 ============
@app.post("/api/prompt/detect", response_model=PromptDetectResponse)
async def detect_prompt(request: PromptDetectRequest):
    """Prompt攻击检测"""
    result = detector.detect(request.text)
    return result

@app.post("/api/output/desensitize", response_model=DesensitizeResponse)
async def desensitize_output(request: DesensitizeRequest):
    """输出脱敏与合规检测"""
    result = desensitizer.desensitize(request.text)
    return result

@app.post("/api/risk/score", response_model=RiskScoreResponse)
async def get_risk_score(request: RiskScoreRequest):
    """用户行为风险评分"""
    result = risk_scorer.predict(request.history)
    return {"user_id": request.user_id, **result}

@app.get("/api/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)