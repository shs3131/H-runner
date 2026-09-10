"""
Hrunner Hello World Example
Demonstrates the minimal structure for an application built with Hrunner.
"""

import sys
import platform

def main():
    print("========================================")
    print("     Hello from Hrunner Application!    ")
    print("========================================")
    print(f"Python Version:   {platform.python_version()}")
    print(f"Platform:         {platform.platform()}")
    print(f"Executable:       {sys.executable}")
    print("Zero bundled Python runtime inside this EXE.")
    print("The runtime is managed centrally by Hrunner!")

if __name__ == '__main__':
    main()
