from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer
import os

model_id = "sentence-transformers/all-MiniLM-L6-v2"
save_dir = "onnx_model"

if not os.path.exists(save_dir):
    os.makedirs(save_dir)

print(f"Exporting {model_id} to ONNX...")
tokenizer = AutoTokenizer.from_pretrained(model_id)
model = ORTModelForFeatureExtraction.from_pretrained(model_id, export=True)

model.save_pretrained(save_dir)
tokenizer.save_pretrained(save_dir)

print(f"Model saved to {save_dir}")
