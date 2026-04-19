import json
import subprocess
from typing import Dict, Any

class BaseTool:
    def execute(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        raise NotImplementedError

class PDFExtractorTool(BaseTool):
    def execute(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        # Implementazione reale: useremmo pymupdf
        return {"text": "Simulated PDF text", "exit_code": 0}

class GeoDistanceTool(BaseTool):
    def execute(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        # Implementazione reale: useremmo geopy
        return {"distance": 42.0, "unit": "km", "exit_code": 0}
