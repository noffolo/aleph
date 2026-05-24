import json
import os
import tempfile
import unittest

from tools import PDFExtractorTool, GeoDistanceTool


class TestPDFExtractorTool(unittest.TestCase):
    def setUp(self):
        self.tool = PDFExtractorTool()

    def test_missing_path(self):
        result = self.tool.execute({"path": ""})
        self.assertEqual(result["exit_code"], 1)
        self.assertIn("required", result["error"])

    def test_nonexistent_file(self):
        result = self.tool.execute({"path": "/nonexistent/file.pdf"})
        self.assertEqual(result["exit_code"], 1)
        self.assertIn("not found", result["error"])

    def test_invalid_file(self):
        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as f:
            f.write(b"not a real pdf")
            tmp_path = f.name

        try:
            result = self.tool.execute({"path": tmp_path})
            self.assertEqual(result["exit_code"], 1)
            self.assertIn("error", result)
        finally:
            os.unlink(tmp_path)


class TestGeoDistanceTool(unittest.TestCase):
    def setUp(self):
        self.tool = GeoDistanceTool()

    def test_missing_args(self):
        result = self.tool.execute({"from": "", "to": ""})
        self.assertEqual(result["exit_code"], 1)
        self.assertIn("required", result["error"])

    def test_rome_to_milan_distance(self):
        result = self.tool.execute({
            "from": {"lat": 41.9028, "lon": 12.4964},
            "to": {"lat": 45.4642, "lon": 9.1900},
        })
        self.assertEqual(result["exit_code"], 0)
        self.assertGreater(result["distance"], 400)
        self.assertLess(result["distance"], 600)
        self.assertEqual(result["unit"], "km")

    def test_paris_to_london_via_string_coords(self):
        result = self.tool.execute({
            "from": "48.8566,2.3522",
            "to": "51.5074,-0.1278",
        })
        self.assertEqual(result["exit_code"], 0)
        self.assertGreater(result["distance"], 300)
        self.assertLess(result["distance"], 500)

    def test_same_point(self):
        result = self.tool.execute({
            "from": {"lat": 0, "lon": 0},
            "to": {"lat": 0, "lon": 0},
        })
        self.assertEqual(result["exit_code"], 0)
        self.assertAlmostEqual(result["distance"], 0, places=2)

    def test_invalid_address(self):
        result = self.tool.execute({
            "from": "this-is-not-a-real-place-12345xyz",
            "to": "42.0,42.0",
        })
        self.assertEqual(result["exit_code"], 1)
        self.assertIn("could not resolve", result["error"])


if __name__ == "__main__":
    unittest.main()
