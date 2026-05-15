import os

import pytest

import config
from heatmap import generate_all_heatmaps, generate_heatmap


class TestGenerateHeatmap:
    def test_creates_png_file(self, tmp_path):
        output = os.path.join(tmp_path, "test.png")
        result = generate_heatmap(
            [(100, 200), (300, 400), (500, 600)],
            output,
            "Test Heatmap",
        )
        assert result == output
        assert os.path.isfile(output)
        assert output.endswith(".png")

    def test_generated_image_has_reasonable_size(self, tmp_path):
        output = os.path.join(tmp_path, "test.png")
        generate_heatmap([(100, 200), (300, 400)], output, "Test")
        size = os.path.getsize(output)
        assert size > 100

    def test_empty_positions_still_produces_output(self, tmp_path):
        output = os.path.join(tmp_path, "empty.png")
        generate_heatmap([], output, "Empty")
        assert os.path.isfile(output)

    def test_single_position_does_not_error(self, tmp_path):
        output = os.path.join(tmp_path, "single.png")
        generate_heatmap([(320, 240)], output, "Single")
        assert os.path.isfile(output)


class TestGenerateAllHeatmaps:
    def test_returns_empty_urls_when_bucket_not_set(self, tmp_path, tracking_history):
        urls = generate_all_heatmaps(
            tracking_history,
            "test_match",
            str(tmp_path),
        )
        assert urls["overall_url"] == ""
        assert urls["defensive_url"] == ""
        assert urls["attacking_url"] == ""

    def test_returns_valid_urls_when_bucket_set(self, tmp_path, tracking_history, monkeypatch):
        monkeypatch.setattr("heatmap._upload_to_s3", lambda fp, b, k: f"https://{b}.s3.us-east-1.amazonaws.com/{k}")
        urls = generate_all_heatmaps(
            tracking_history,
            "test_match",
            str(tmp_path),
            bucket="test-bucket",
        )
        assert "overall_url" in urls
        assert urls["overall_url"] != ""

    def test_creates_three_heatmap_files(self, tmp_path, tracking_history):
        generate_all_heatmaps(tracking_history, "test_match", str(tmp_path))
        files = os.listdir(tmp_path)
        pngs = [f for f in files if f.endswith(".png")]
        assert len(pngs) == 3

    def test_empty_history_does_not_error(self, tmp_path):
        urls = generate_all_heatmaps([], "test_match", str(tmp_path))
        assert isinstance(urls, dict)
