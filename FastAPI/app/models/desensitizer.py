# desensitizer.py
"""输出脱敏：正则掩码 + SpaCy NER + 违规话术合规检测。

SpaCy 或其中文模型未安装时自动跳过 NER 层，保留正则与关键词合规检测。
正则层对重叠实体做去重（长实体优先），NER 层带常见误报词黑名单。
"""

import logging
import re
from typing import List, Tuple

try:
    import spacy

    SPACY_AVAILABLE = True
except Exception:
    spacy = None
    SPACY_AVAILABLE = False

logger = logging.getLogger("desensitizer")


class Desensitizer:
    def __init__(self):
        # 正则模式
        self.patterns = {
            "phone": re.compile(r"1[3-9]\d{9}"),
            "id_card": re.compile(
                r"[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[0-9Xx]"
            ),
            "bank_card": re.compile(r"\d{16,19}"),
            "email": re.compile(
                r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b"
            ),
        }

        # 违规理财话术关键词
        self.illegal_keywords = [
            "保本保息",
            "承诺收益",
            "绝对赚钱",
            "稳赚不赔",
            "内部消息",
            "坐庄",
            "拉升",
            "涨停",
            "无风险",
            "高收益",
            "翻倍",
        ]

        # 加载 SpaCy 中文 NER 模型（缺失时跳过 NER 层）
        self.nlp = None
        if SPACY_AVAILABLE:
            try:
                self.nlp = spacy.load("zh_core_web_sm")
            except Exception as exc:
                logger.warning("SpaCy 中文模型未就绪，跳过 NER 层: %s", exc)
        else:
            logger.warning("未安装 spacy，跳过 NER 层")

    def mask_text(self, text: str, entity_type: str, span: Tuple[int, int]) -> str:
        """对指定区间进行掩码"""
        original = text[span[0] : span[1]]
        if entity_type == "phone":
            masked = original[:3] + "****" + original[-4:]
        elif entity_type == "id_card":
            masked = original[:6] + "********" + original[-4:]
        elif entity_type == "bank_card":
            masked = "****" + original[-4:]
        else:
            masked = "***"
        return text[: span[0]] + masked + text[span[1] :]

    def regex_desensitize(self, text: str) -> Tuple[str, List[dict]]:
        """正则层脱敏（长实体优先，去除重叠实体）"""
        entities = []
        for entity_type, pattern in self.patterns.items():
            for match in pattern.finditer(text):
                entities.append(
                    {
                        "type": entity_type,
                        "span": (match.start(), match.end()),
                        "value": match.group(),
                    }
                )

        # 按 span 长度降序，长实体优先（身份证 > 银行卡等）
        entities.sort(key=lambda x: (x["span"][1] - x["span"][0]), reverse=True)

        # 去重：保留不与已选实体重叠的
        filtered = []
        for ent in entities:
            overlap = any(
                not (ent["span"][1] <= kept["span"][0] or ent["span"][0] >= kept["span"][1])
                for kept in filtered
            )
            if not overlap:
                filtered.append(ent)

        # 从后往前替换，避免偏移
        for ent in sorted(filtered, key=lambda x: x["span"][0], reverse=True):
            text = self.mask_text(text, ent["type"], ent["span"])
        return text, filtered

    def ner_desensitize(self, text: str) -> Tuple[str, List[dict]]:
        """NER 层脱敏（人名、机构等；模型未就绪时跳过）"""
        if self.nlp is None:
            return text, []

        # 常见误报词黑名单
        blacklist = {"手机号", "身份证", "银行卡", "密码", "账号", "姓名"}
        doc = self.nlp(text)
        entities = []
        for ent in doc.ents:
            if ent.label_ in ["PERSON", "ORG", "GPE", "LOC"] and ent.text not in blacklist:
                entities.append(
                    {
                        "type": ent.label_,
                        "span": (ent.start_char, ent.end_char),
                        "value": ent.text,
                    }
                )
        for ent in sorted(entities, key=lambda x: x["span"][0], reverse=True):
            text = self.mask_text(text, "ner", ent["span"])
        return text, entities

    def compliance_check(self, text: str) -> dict:
        """合规评分：检测违规话术"""
        violations = []
        for keyword in self.illegal_keywords:
            if keyword in text:
                violations.append(keyword)
        score = 100 - len(violations) * 10  # 每个违规词扣10分
        return {
            "score": max(0, score),
            "violations": violations,
            "is_compliant": len(violations) == 0,
        }

    def desensitize(self, text: str) -> dict:
        """完整脱敏流程"""
        # 1. 正则脱敏
        text, regex_entities = self.regex_desensitize(text)
        # 2. NER 脱敏
        text, ner_entities = self.ner_desensitize(text)
        # 3. 合规检查
        compliance = self.compliance_check(text)

        return {
            "desensitized_text": text,
            "detected_entities": regex_entities + ner_entities,
            "compliance": compliance,
        }
