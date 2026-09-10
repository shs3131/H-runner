"""
Hrunner Vision Camera Application
Modern, high-performance desktop camera application with real-time video filters,
HUD overlay, and snapshot capture using OpenCV, NumPy, and Pillow.
"""

import os
import sys
import time
import math
from datetime import datetime
import cv2
import numpy as np
from PIL import Image, ImageDraw, ImageFont

FILTERS = {
    1: "NORMAL",
    2: "SEPIA",
    3: "CANNY EDGE",
    4: "CYBERPUNK",
    5: "GAUSSIAN BLUR",
}

def apply_filter(frame, mode):
    if mode == 1:
        return frame
    elif mode == 2:
        # Sepia transformation matrix
        kernel = np.array([[0.272, 0.534, 0.131],
                           [0.349, 0.686, 0.168],
                           [0.393, 0.769, 0.189]])
        sepia = cv2.transform(frame, kernel)
        return np.clip(sepia, 0, 255).astype(np.uint8)
    elif mode == 3:
        # Canny edge detection
        gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
        edges = cv2.Canny(gray, 50, 150)
        return cv2.cvtColor(edges, cv2.COLOR_GRAY2BGR)
    elif mode == 4:
        # Cyberpunk / Neon Invert
        inv = cv2.bitwise_not(frame)
        hsv = cv2.cvtColor(inv, cv2.COLOR_BGR2HSV)
        hsv[:, :, 0] = (hsv[:, :, 0] + 50) % 180
        return cv2.cvtColor(hsv, cv2.COLOR_HSV2BGR)
    elif mode == 5:
        # Gaussian blur
        return cv2.GaussianBlur(frame, (21, 21), 0)
    return frame

def draw_synthetic_frame(t, w=640, h=480):
    """Draw an animated high-tech test pattern when no physical webcam is present."""
    y_coords = np.linspace(0, 1, h)[:, None]
    x_coords = np.linspace(0, 1, w)[None, :]
    
    b = np.clip(25 + 20 * np.sin(x_coords * 4 + t), 0, 255).astype(np.uint8)
    g = np.clip(30 + 35 * y_coords + 15 * np.cos(t * 1.5), 0, 255).astype(np.uint8)
    r = np.clip(40 + 20 * np.sin(t * 2), 0, 255).astype(np.uint8)
    
    img = np.zeros((h, w, 3), dtype=np.uint8)
    img[:, :, 0] = b
    img[:, :, 1] = g
    img[:, :, 2] = r

    # Animated radar sweep
    cx, cy = w // 2, h // 2
    cv2.circle(img, (cx, cy), 140, (0, 180, 255), 1, cv2.LINE_AA)
    cv2.circle(img, (cx, cy), 90, (0, 130, 200), 1, cv2.LINE_AA)
    cv2.circle(img, (cx, cy), 40, (0, 90, 150), 1, cv2.LINE_AA)

    angle = (t * 2) % (2 * math.pi)
    end_x = int(cx + 140 * math.cos(angle))
    end_y = int(cy + 140 * math.sin(angle))
    cv2.line(img, (cx, cy), (end_x, end_y), (0, 255, 200), 2, cv2.LINE_AA)

    # Crosshairs
    cv2.line(img, (cx - 160, cy), (cx + 160, cy), (70, 70, 70), 1)
    cv2.line(img, (cx, cy - 160), (cx, cy + 160), (70, 70, 70), 1)

    # Simulated tracking bounding box
    box_x = int(cx - 70 + 30 * math.sin(t * 1.8))
    box_y = int(cy - 60 + 20 * math.cos(t * 1.4))
    cv2.rectangle(img, (box_x, box_y), (box_x + 140, box_y + 120), (0, 255, 128), 2, cv2.LINE_AA)
    cv2.putText(img, "TARGET LOCKED", (box_x, box_y - 8), cv2.FONT_HERSHEY_SIMPLEX, 0.45, (0, 255, 128), 1, cv2.LINE_AA)

    return img

def render_hud(frame, filter_mode, fps, snapshot_count, is_synthetic, flash_frames):
    h, w = frame.shape[:2]

    # Top translucent HUD bar
    overlay = frame.copy()
    cv2.rectangle(overlay, (0, 0), (w, 54), (15, 15, 20), -1)
    # Bottom HUD bar
    cv2.rectangle(overlay, (0, h - 38), (w, h), (15, 15, 20), -1)
    cv2.addWeighted(overlay, 0.75, frame, 0.25, 0, frame)

    # Accent bar
    cv2.line(frame, (0, 54), (w, 54), (0, 160, 255), 2)
    cv2.line(frame, (0, h - 38), (w, h - 38), (50, 50, 60), 1)

    # Title & Telemetry
    title = "HRUNNER VISION CAM"
    cv2.putText(frame, title, (16, 32), cv2.FONT_HERSHEY_DUPLEX, 0.7, (255, 255, 255), 1, cv2.LINE_AA)
    
    status_text = "SIMULATED SENSOR" if is_synthetic else "LIVE HARDWARE CAM"
    status_color = (0, 200, 255) if is_synthetic else (0, 255, 128)
    cv2.putText(frame, status_text, (280, 24), cv2.FONT_HERSHEY_SIMPLEX, 0.42, status_color, 1, cv2.LINE_AA)
    
    telemetry = f"FPS: {fps:.1f} | {w}x{h} | SNAPS: {snapshot_count}"
    cv2.putText(frame, telemetry, (280, 44), cv2.FONT_HERSHEY_SIMPLEX, 0.42, (180, 180, 180), 1, cv2.LINE_AA)

    # Filter Badge (Top Right)
    filter_label = f"[{filter_mode}] {FILTERS.get(filter_mode, 'UNKNOWN')}"
    (tw, th), _ = cv2.getTextSize(filter_label, cv2.FONT_HERSHEY_SIMPLEX, 0.5, 1)
    badge_x = w - tw - 24
    cv2.rectangle(frame, (badge_x - 8, 14), (w - 12, 42), (0, 103, 192), -1)
    cv2.putText(frame, filter_label, (badge_x, 32), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (255, 255, 255), 1, cv2.LINE_AA)

    # Bottom Instructions Bar
    help_text = "[1-5] Switch Filter   [SPACE] Save Snapshot   [Q/ESC] Quit"
    cv2.putText(frame, help_text, (16, h - 14), cv2.FONT_HERSHEY_SIMPLEX, 0.44, (200, 200, 200), 1, cv2.LINE_AA)

    # Flash effect on snapshot
    if flash_frames > 0:
        flash_alpha = min(1.0, flash_frames / 4.0)
        flash_layer = np.full_like(frame, 255)
        cv2.addWeighted(flash_layer, flash_alpha * 0.7, frame, 1.0 - flash_alpha * 0.7, 0, frame)

    return frame

def main():
    print("=" * 50)
    print("      Hrunner Vision Camera Application")
    print("=" * 50)
    print(f"Python: {sys.version.split()[0]}")
    print(f"OpenCV: {cv2.__version__}")
    print(f"NumPy:  {np.__version__}")
    print("=" * 50)

    # Try opening physical camera
    cap = cv2.VideoCapture(0, cv2.CAP_DSHOW)
    is_synthetic = False
    if not cap.isOpened():
        print("Note: Physical webcam not detected. Engaging synthetic test pattern.")
        is_synthetic = True
    else:
        cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
        cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

    window_name = "Hrunner Vision Camera"
    cv2.namedWindow(window_name, cv2.WINDOW_AUTOSIZE)

    filter_mode = 1
    snapshot_count = 0
    flash_frames = 0
    prev_time = time.time()
    fps = 30.0
    start_time = time.time()

    # Headless timeout check (for automated CLI / smoke tests)
    max_frames = int(os.getenv("CAMERA_APP_MAX_FRAMES", "0"))
    frame_idx = 0

    try:
        while True:
            cur_time = time.time()
            dt = cur_time - prev_time
            prev_time = cur_time
            if dt > 0:
                fps = 0.9 * fps + 0.1 * (1.0 / dt)

            t = cur_time - start_time

            if is_synthetic:
                raw_frame = draw_synthetic_frame(t)
            else:
                ret, raw_frame = cap.read()
                if not ret:
                    raw_frame = draw_synthetic_frame(t)
                    is_synthetic = True

            # Process filter
            filtered = apply_filter(raw_frame, filter_mode)

            # Render UI HUD
            display_frame = render_hud(filtered, filter_mode, fps, snapshot_count, is_synthetic, flash_frames)
            if flash_frames > 0:
                flash_frames -= 1

            cv2.imshow(window_name, display_frame)

            key = cv2.waitKey(15) & 0xFF
            if key in [ord('q'), ord('Q'), 27]: # 27 = ESC
                break
            elif key in [ord('1'), ord('2'), ord('3'), ord('4'), ord('5')]:
                filter_mode = key - ord('0')
            elif key == ord(' ') or key == ord('s') or key == ord('S'):
                snapshot_count += 1
                flash_frames = 5
                
                rgb = cv2.cvtColor(raw_frame, cv2.COLOR_BGR2RGB)
                pil_img = Image.fromarray(rgb)
                
                snap_name = f"snapshot_{datetime.now().strftime('%Y%m%d_%H%M%S')}.png"
                pil_img.save(snap_name)
                print(f"[Snapshot] Saved {snap_name} via Pillow ({pil_img.size[0]}x{pil_img.size[1]})")

            frame_idx += 1
            if max_frames > 0 and frame_idx >= max_frames:
                print(f"Reached automated test limit of {max_frames} frames. Exiting cleanly.")
                break

    finally:
        if not is_synthetic:
            cap.release()
        cv2.destroyAllWindows()
        print("Camera application closed successfully.")

if __name__ == "__main__":
    main()
