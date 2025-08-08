import time

def main():
    end = time.time() + 10
    while time.time() < end:
        print("hello")
        time.sleep(0.1)  # adjust if you want faster/slower printing

if __name__ == "__main__":
    main()