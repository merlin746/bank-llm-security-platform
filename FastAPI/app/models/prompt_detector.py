# prompt_detector.py
"""Prompt 攻击检测器。

优先加载 Qwen2(LoRA) + 二分类头做语义检测；深度学习依赖或模型产物缺失时
自动降级为规则层，保证 Demo 环境（模型未训练）也能启动服务。
推理默认 float32，CPU / GPU 均兼容（集成显卡环境亦可运行）。
"""

import logging
import os
import re

try:
    import torch
    from peft import PeftModel
    from transformers import AutoModelForCausalLM, AutoTokenizer

    HF_AVAILABLE = True
except Exception:  # 未安装 torch / transformers / peft 时降级
    torch = None
    HF_AVAILABLE = False

logger = logging.getLogger("prompt_detector")


class PromptDetector:
    def __init__(self, base_model_path: str, adapter_path: str):
        self.base_model_path = base_model_path
        self.adapter_path = adapter_path
        self.model = None
        self.classifier = None
        self.tokenizer = None
        self.model_ready = False
        self.degradation_reason = "模型层未就绪，本次仅执行规则检测"

        # 识别攻击动作及其目标，避免将银行卡查询、密码重置等正常业务当作攻击。
        self.attack_rules = [
            ("覆盖已有指令", r"(?:ignore|forget|disregard).{0,80}?(?:previous|above|system)"),
            ("覆盖已有指令", r"(?:忽略|忘记|无视|抛弃|不遵守).{0,20}?(?:之前|先前|以上|上述|原有|所有|系统|安全).{0,12}?(?:指令|要求|规则|限制|约束|提示词)"),
            ("不受限制的角色伪装", r"you are now (?:acting as|扮演)"),
            ("不受限制的角色伪装", r"(?:你(?:现在)?是|扮演|充当).{0,16}?(?:\bDAN\b|不受(?:任何)?限制|无限制|无约束)"),
            ("解除安全限制", r"jailbreak|越狱|突破限制|(?:解除|取消|关闭).{0,12}?(?:安全限制|安全约束|安全策略)|(?:无限制|无约束).{0,8}?(?:回答|执行)"),
            ("绕过权限控制", r"(?:绕过|跳过|规避|禁用).{0,12}?(?:权限|鉴权|访问控制|安全校验|身份认证)|bypass.{0,30}?(?:permission|authorization|access control|security)"),
            ("泄露内部安全配置", r"(?:泄露|透露|暴露).{0,20}?(?:系统|内部).{0,20}?(?:安全策略|策略配置|提示词|指令|配置)"),
            ("提取系统提示词", r"(?:输出|打印|显示|提供|复述).{0,16}?(?:系统提示词|系统指令|内部提示词|内部指令)|what (?:are|is) your (?:system|internal) (?:prompt|instruction)|(?:reveal|print|show).{0,30}?(?:system prompt|internal instructions?)"),
            ("窃取敏感信息", r"extract.{0,60}?(?:password|secret|key|credential)|(?:泄露|窃取|盗取|套取).{0,24}?(?:密码|口令|密钥|凭证|客户数据|客户资料|身份证|银行卡号)"),
        ]
        # 最短匹配保留后续攻击动作的检查机会，避免被前面的否定要求吞并。
        self.attack_rules = [
            (reason, re.compile(pattern, re.IGNORECASE | re.DOTALL))
            for reason, pattern in self.attack_rules
        ]

        if not HF_AVAILABLE:
            self.degradation_reason = "深度学习依赖未安装或不可用，本次仅执行规则检测"
            logger.warning("深度学习依赖缺失，PromptDetector 仅规则层可用")
            return

        classifier_path = os.path.join(adapter_path, "classifier_head.pt")
        if not os.path.isdir(adapter_path) or not os.path.isfile(classifier_path):
            self.degradation_reason = "模型适配器或分类头未就绪，本次仅执行规则检测"
            logger.warning("PromptDetector 模型产物缺失，仅规则层可用")
            return

        try:
            self.tokenizer = AutoTokenizer.from_pretrained(
                base_model_path, trust_remote_code=True
            )
            if self.tokenizer.pad_token is None:
                self.tokenizer.pad_token = self.tokenizer.eos_token

            # float32：CPU / GPU 均兼容
            self.base_model = AutoModelForCausalLM.from_pretrained(
                base_model_path,
                torch_dtype=torch.float32,
                trust_remote_code=True,
            )
            # 加载 LoRA 适配器（由 scripts/train_prompt_safety.py 生成）
            self.model = PeftModel.from_pretrained(self.base_model, adapter_path)
            self.model.eval()

            # 加载二分类头（与训练脚本保存的结构一致）
            hidden_size = self.base_model.config.hidden_size
            self.classifier = torch.nn.Linear(hidden_size, 2)
            self.classifier.load_state_dict(torch.load(classifier_path, map_location="cpu"))
            self.classifier.eval()
            self.model_ready = True
            self.degradation_reason = ""
            logger.info("PromptDetector 模型层加载完成")
        except Exception as exc:  # 模型文件缺失 / 加载失败等
            self.model = None
            self.degradation_reason = "模型层加载失败，本次仅执行规则检测"
            logger.warning("PromptDetector 模型层加载失败（%s），仅规则层可用", exc)

    def rule_check(self, text: str) -> tuple[bool, str]:
        """规则层快速检测"""
        for reason, pattern in self.attack_rules:
            for match in pattern.finditer(text):
                # “不要绕过权限”“如何防止越狱”等防护要求不属于攻击指令。
                prefix = re.split(r"[，。！？；,;.!?\n]", text[max(0, match.start() - 24):match.start()])[-1]
                if re.search(
                    r"(?:不要|不得|禁止|不应|不能|避免|防止|防范)"
                    r"(?:(?!但是|但|然而|不过|却|然后|改为|现在|请).){0,20}$",
                    prefix,
                ):
                    continue
                return True, f"命中规则：{reason}"
        return False, ""

    def detection_metadata(self, mode: str, degradation_reason: str = None) -> dict:
        """模式表示实际使用的检测层；降级表示语义模型本次不可用。"""
        reason = degradation_reason
        if reason is None:
            reason = "" if self.model_ready else self.degradation_reason
        return {
            "detection_mode": mode,
            "degraded": bool(reason),
            "degradation_reason": reason,
        }

    def model_check(self, text: str) -> tuple[bool, float]:
        """模型层深度检测（模型未就绪时返回安全）"""
        if not self.model_ready:
            return False, 0.0
        inputs = self.tokenizer(
            text, return_tensors="pt", truncation=True, max_length=512
        )
        inputs = {k: v.to(self.model.device) for k, v in inputs.items()}
        with torch.no_grad():
            outputs = self.model(**inputs, output_hidden_states=True)
            hidden = outputs.hidden_states[-1]  # 最后一层
            # 取最后一个有效 token（非 padding）
            attention_mask = inputs["attention_mask"]
            last_idx = attention_mask.sum(dim=1) - 1
            last_hidden = hidden[torch.arange(hidden.size(0)), last_idx]
            logits = self.classifier(last_hidden)
            probs = torch.softmax(logits, dim=-1)
            attack_prob = probs[0][1].item()  # 攻击概率
        return attack_prob > 0.5, attack_prob

    def detect(self, text: str) -> dict:
        """完整检测流程"""
        # 先走规则层
        is_attack, reason = self.rule_check(text)
        if is_attack:
            return {
                "is_attack": True,
                "confidence": 1.0,
                "reason": reason,
                "layer": "rule",
                **self.detection_metadata("rules"),
            }

        # 规则未命中且模型不可用：仅规则层
        if not self.model_ready:
            return {
                "is_attack": False,
                "confidence": 0.0,
                "reason": "规则层未发现攻击特征",
                "layer": "rule",
                **self.detection_metadata("rules"),
            }

        try:
            is_attack, confidence = self.model_check(text)
        except Exception as exc:
            logger.warning("PromptDetector 模型推理失败（%s），本次仅规则层检测", exc)
            return {
                "is_attack": False,
                "confidence": 0.0,
                "reason": "规则层未发现攻击特征",
                "layer": "rule",
                **self.detection_metadata("rules", "模型层推理失败，本次仅执行规则检测"),
            }
        return {
            "is_attack": is_attack,
            "confidence": confidence,
            "reason": "模型语义检测" if is_attack else "安全",
            "layer": "model",
            **self.detection_metadata("model"),
        }
