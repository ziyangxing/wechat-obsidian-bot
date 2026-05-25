"""WeChat voice transcription using whisper.
Usage: python transcribe_voice.py <voice_file_path>
Outputs JSON with transcript text to stdout.
"""
import sys
import json
import subprocess
import os

# Fix Windows GBK encoding issue
sys.stdout.reconfigure(encoding='utf-8', errors='replace')


def transcribe(file_path: str) -> dict:
    if not os.path.exists(file_path):
        return {"error": f"File not found: {file_path}", "text": ""}

    # Convert to WAV using ffmpeg
    wav_path = file_path + ".wav"
    try:
        subprocess.run(
            ["ffmpeg", "-y", "-i", file_path, "-ar", "16000", "-ac", "1", wav_path],
            capture_output=True,
            timeout=120,
            check=True,
        )
    except subprocess.CalledProcessError as e:
        return {"error": f"FFmpeg conversion failed: {e.stderr.decode(errors='replace')[:200]}", "text": ""}
    except subprocess.TimeoutExpired:
        return {"error": "FFmpeg conversion timed out", "text": ""}

    # Run whisper
    try:
        import whisper
        model = whisper.load_model("small")  # small is much better for Chinese
        result = model.transcribe(wav_path, language="zh", verbose=False)
        text = result["text"].strip()
    except Exception as e:
        return {"error": f"Whisper transcription failed: {e}", "text": ""}
    finally:
        # Clean up WAV file
        if os.path.exists(wav_path):
            os.remove(wav_path)

    return {"text": text, "error": ""}


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No file path provided", "text": ""}))
        sys.exit(1)

    result = transcribe(sys.argv[1])
    print(json.dumps(result, ensure_ascii=False))
