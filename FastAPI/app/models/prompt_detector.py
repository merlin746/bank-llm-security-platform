# prompt_detector.py
import re
import torch
from transformers import AutoModelForSequenceClassification, AutoTokenizer
from peft import PeftModel

class PromptDetector:
    def __init__(self, base_model_path: str, adapter_path: str):
        # 加载基座模型 + LoRA适配器
        self.tokenizer = AutoTokenizer.from_pretrained(base_model_path)
        self.model = AutoModelForSequenceClassification.from_pretrained(
            base_model_path, 
            num_labels=2  # 二分类: safe / attack
        )
        self.model = PeftModel.from_pretrained(self.model, adapter_path)
        self.model.eval()
        
        # 规则库：常见越狱/注入模式
        self.attack_patterns = [
            r"(?i)(ignore|forget| disregard).*(previous|above|system)",
            r"(?i)you are now (acting as|扮演)",
            r"(?i)jailbreak|越狱|突破限制",
            r"(?i)extract.*(password|secret|key|credential)",
            r"(?i)what (are|is) your (system|internal) (prompt|instruction)",
            r"(?i)输出.*(身份证|银行卡|密码)",
        ]
    
    def rule_check(self, text: str) -> tuple[bool, str]:
        """规则层快速检测"""
        for pattern in self.attack_patterns:
            if re.search(pattern, text):
                return True, f"命中规则: {pattern}"
        return False, ""
    
    def model_check(self, text: str) -> tuple[bool, float]:
        """模型层深度检测"""
        inputs = self.tokenizer(text, return_tensors="pt", truncation=True, max_length=512)
        with torch.no_grad():
            outputs = self.model(**inputs)
            probs = torch.softmax(outputs.logits, dim=-1)
            attack_prob = probs[0][1].item()  # 攻击概率
        return attack_prob > 0.5, attack_prob
    
    def detect(self, text: str) -> dict:
        """完整检测流程"""
        # 先走规则层
        is_attack, reason = self.rule_check(text)
        if is_attack:
            return {"is_attack": True, "confidence": 1.0, "reason": reason, "layer": "rule"}
        
        # 规则未命中，走模型层
        is_attack, confidence = self.model_check(text)
        return {
            "is_attack": is_attack,
            "confidence": confidence,
            "reason": "模型语义检测" if is_attack else "安全",
            "layer": "model"
        }
