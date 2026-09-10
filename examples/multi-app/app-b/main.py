"""
Multi-App Example: Application B
Uses shared 'requests 2.32.5' and 'numpy 2.3.2'.
"""

import requests
import numpy as np

def main():
    print("========================================")
    print("           Application B                ")
    print("========================================")
    print(f"Requests version: {requests.__version__} (SHARED)")
    print(f"NumPy version:    {np.__version__} (ISOLATED TO APP B)")
    print(f"NumPy array test: {np.ones((2, 2))}")

if __name__ == '__main__':
    main()
