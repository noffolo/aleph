import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from main import analyze_sentiment_simple


class TestSentiment:
    def test_positive(self):
        score, label = analyze_sentiment_simple("ottima crescita eccellente")
        assert label == "positive"
        assert score > 0

    def test_negative(self):
        score, label = analyze_sentiment_simple("pessimo fallimento crisi")
        assert label == "negative"
        assert score < 0

    def test_neutral(self):
        score, label = analyze_sentiment_simple("il tavolo e di legno")
        assert label == "neutral"
        assert -0.2 <= score <= 0.2

    def test_empty(self):
        score, label = analyze_sentiment_simple("")
        assert label == "neutral"
        assert score == 0.0

    def test_mixed(self):
        score, label = analyze_sentiment_simple("good growth but high risk")
        assert label in ("positive", "negative", "neutral")
