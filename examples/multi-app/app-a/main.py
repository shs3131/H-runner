"""
Multi-App Example: Application A
Uses shared 'requests 2.32.5' and 'numpy 1.26.4'.
"""

import requests
import numpy as np

def main():
    print("========================================")
    print("           Application A                ")
    print("========================================")
    print(f"Requests version: {requests.__version__} (SHARED)")
    print(f"NumPy version:    {np.__version__} (ISOLATED TO APP A)")
    print(f"NumPy array test: {np.array([1, 2, 3]) * 2}")

if __name__ == '__main__':
    main()
