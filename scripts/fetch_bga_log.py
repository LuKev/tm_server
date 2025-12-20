import argparse
import time
import os
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def main():
    parser = argparse.ArgumentParser(description='Fetch BGA game logs.')
    parser.add_argument('table_id', help='The BGA table ID (e.g., 713277654)')
    parser.add_argument('--output', '-o', default='bga_log.txt', help='Output file path')
    parser.add_argument('--user-data-dir', help='Path to Chrome user data dir for persistent login')
    args = parser.parse_args()

    url = f"https://boardgamearena.com/gamereview?table={args.table_id}"
    print(f"Fetching logs for table {args.table_id} from {url}...")

    chrome_options = Options()
    if args.user_data_dir:
        chrome_options.add_argument(f"user-data-dir={args.user_data_dir}")
    else:
        # If no user data dir, we might need manual login
        print("No user-data-dir provided. You may need to log in manually.")

    driver = webdriver.Chrome(options=chrome_options)

    try:
        driver.get(url)

        # Wait for user to log in if needed
        print("Waiting for page to load... (If redirected to login, please log in)")
        
        # Wait for the log container to be present. 
        # BGA logs are often in a div with id 'logs' or class 'logs'.
        # We'll try to find the main log container.
        # In replays, it's often 'gamelogs' or similar.
        # We'll wait for a generic element that indicates the game loaded.
        WebDriverWait(driver, 300).until(
            EC.presence_of_element_located((By.ID, "gamelogs")) 
        )
        
        print("Game log container found. Extracting text...")
        
        # Allow some time for dynamic content to load
        time.sleep(5) 
        
        # Extract text from the log container
        log_element = driver.find_element(By.ID, "gamelogs")
        log_text = log_element.text
        
        # Save to file
        with open(args.output, 'w', encoding='utf-8') as f:
            f.write(log_text)
            
        print(f"Successfully saved {len(log_text)} characters to {args.output}")

    except Exception as e:
        print(f"Error: {e}")
        # Dump page source for debugging if needed
        # with open("debug_page_source.html", "w") as f:
        #     f.write(driver.page_source)
    finally:
        driver.quit()

if __name__ == "__main__":
    main()
