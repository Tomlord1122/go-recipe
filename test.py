import time
import sys

def main():
    print("開始執行腳本...")
    try:
        # 睡眠 5 秒鐘
        print("程式將休眠 5 秒鐘...")
        time.sleep(5)
        print("休眠完成！")
    except KeyboardInterrupt:
        print("\n程式被使用者中斷")
    except Exception as e:
        print(f"發生錯誤: {e}")
    finally:
        print("程式正在終止...")
        sys.exit(0)

if __name__ == "__main__":
    main()
