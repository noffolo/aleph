import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from main import analyze_sentiment_simple


class TestSentimentHeuristic:
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

    def test_returns_uncalibrated_scores(self):
        score, label = analyze_sentiment_simple("excellent great positive")
        assert -1.0 <= score <= 1.0
        assert label in ("positive", "negative", "neutral")

    def test_bilingual_italian(self):
        score, label = analyze_sentiment_simple("buono successo crescita")
        assert label == "positive"
        assert score > 0

    def test_bilingual_english(self):
        score, label = analyze_sentiment_simple("bad terrible decline")
        assert label == "negative"
        assert score < 0