# Start 10 process to run the test.sh and wait all of them to finish

import os
import subprocess
import time
import sys

def run():
    # Run the test.sh
    # Change the path to your test.sh
    subprocess.call(["./send.sh"])

def main():
    # Start 10 process to run the test.sh
    for i in range(10):
        pid = os.fork()
        if pid == 0:
            run()
            sys.exit(0)
    # Wait all of them to finish
    for i in range(10):
        os.waitpid(-1, 0)


main()
