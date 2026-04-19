import torch
from torch_geometric.nn import GCNConv
from torch_geometric.data import Data
import torch.nn.functional as F

class LinkPredictor(torch.nn.Module):
    def __init__(self, in_channels, hidden_channels, out_channels):
        super(LinkPredictor, self).__init__()
        self.conv1 = GCNConv(in_channels, hidden_channels)
        self.conv2 = GCNConv(hidden_channels, out_channels)

    def encode(self, x, edge_index):
        x = self.conv1(x, edge_index).relu()
        return self.conv2(x, edge_index)

    def decode(self, z, edge_label_index):
        # Prodotto scalare per determinare la probabilità di un link
        return (z[edge_label_index[0]] * z[edge_label_index[1]]).sum(dim=-1)

def train_link_prediction(num_nodes, edge_index, features):
    """
    Esempio di training loop per Link Prediction su Aleph Graph
    """
    model = LinkPredictor(features.shape[1], 128, 64)
    optimizer = torch.optim.Adam(model.parameters(), lr=0.01)
    
    # Label: 1 per i link esistenti, 0 per i link campionati negativamente
    # (Semplificazione per l'architettura Aleph)
    
    model.train()
    for epoch in range(1, 21):
        optimizer.zero_grad()
        z = model.encode(features, edge_index)
        
        # Loss calcolata sui link esistenti
        out = model.decode(z, edge_index)
        loss = F.binary_cross_entropy_with_logits(out, torch.ones(out.size(0)))
        
        loss.backward()
        optimizer.step()
        if epoch % 5 == 0:
            print(f"Epoch: {epoch:03d}, Loss: {loss:.4f}")
            
    return model, z

if __name__ == "__main__":
    print("[NLP] GNN Link Predictor initialized.")
    # In una pipeline reale, i dati verrebbero letti da DuckDB via gRPC
