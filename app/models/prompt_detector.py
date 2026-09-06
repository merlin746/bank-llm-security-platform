# prompt_detector.py
import re
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
import os

class PromptDetector:
    def __init__(self, base_model_path: str, adapter_path: str):
        self.tokenizer = AutoTokenizer.from_pretrained(base_model_path, trust_remote_code=True)
        self.tokenizer.pad_token = self.tokenizer.eos_token
        
        # 加载基座模型（因果LM）
        self.base_model = AutoModelForCausalLM.from_pretrained(
            base_model_path,
            torch_dtype=torch.float16,
            device_map="auto",
            trust_remote_code=True
        )
        # 加载 LoRA 适配器
        self.model = PeftModel.from_pretrained(self.base_model, adapter_path)
        self.model.eval()
        
        # 加载分类头
        classifier_path = os.path.join(adapter_path, "classifier_head.pt")
        hidden_size = self.base_model.config.hidden_size
        self.classifier = torch.nn.Linear(hidden_size, 2, bias=False)
        self.classifier.load_state_dict(torch.load(classifier_path, map_location="cpu"))
        self.classifier.to(self.model.device)
        self.classifier.eval()
        
        # 规则库（后续可配置化）
        self.attack_patterns = [
            r"(?i)(ignore|forget| disregard).*(previous|above|system)",
            r"(?i)you are now (acting as|扮演)",
            r"(?i)jailbreak|越狱|突破限制",
            r"(?i)extract.*(password|secret|key|credential)",
            r"(?i)what (are|is) your (system|internal) (prompt|instruction)",
            r"(?i)输出.*(身份证|银行卡|密码)",
        ]
    
    def rule_check(self, text: str) -> tuple[bool, str]:
        for pattern in self.attack_patterns:
            if re.search(pattern, text, re.IGNORECASE):
                return True, f"命中规则: {pattern}"
        return False, ""
    
    def model_check(self, text: str) -> tuple[bool, float]:
        inputs = self.tokenizer(text, return_tensors="pt", truncation=True, max_length=512)
        # 移动到模型设备
        inputs = {k: v.to(self.model.device) for k, v in inputs.items()}
        with torch.no_grad():
            outputs = self.model(**inputs, output_hidden_states=True)
            hidden_states = outputs.hidden_states[-1]  # 最后一层
            # 取最后一个有效 token（非 padding）
            attention_mask = inputs["attention_mask"]
            last_token_indices = attention_mask.sum(dim=1) - 1
            batch_size = hidden_states.size(0)
            last_hidden = hidden_states[torch.arange(batch_size), last_token_indices]
            logits = self.classifier(last_hidden)
            probs = torch.softmax(logits, dim=-1)
            attack_prob = probs[0][1].item()
        return attack_prob > 0.5, attack_prob
    
    def detect(self, text: str) -> dict:
        is_attack, reason = self.rule_check(text)
        if is_attack:
            return {"is_attack": True, "confidence": 1.0, "reason": reason, "layer": "rule"}
        is_attack, confidence = self.model_check(text)
        return {
            "is_attack": is_attack,
            "confidence": confidence,
            "reason": "模型语义检测" if is_attack else "安全",
            "layer": "model"
        }