"""
Hrunner Requirements Example
Demonstrates managing third-party dependencies from PyPI via requirements.txt.
"""

import sys
import requests

def main():
    print("========================================")
    print("    Hrunner Third-Party Package Demo    ")
    print("========================================")
    print(f"Requests version: {requests.__version__}")
    print(f"Loaded from:      {requests.__file__}")

    # Make a quick request to verify network and SSL certificates
    try:
        r = requests.get("https://httpbin.org/get", timeout=5)
        print(f"HTTP GET status:  {r.status_code}")
        print("Dependency successfully loaded from Hrunner shared package pool!")
    except Exception as e:
        print(f"Request test: {e}")

if __name__ == '__main__':
    main()
