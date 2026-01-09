import argparse
import time
import os
import re
from bs4 import BeautifulSoup
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def parse_log_html(html_content):
    soup = BeautifulSoup(html_content, 'html.parser')

    # Helper to replace amount divs
    def replace_amount(class_name, unit_name):
        for div in soup.find_all("div", class_=class_name):
            amount = div.get_text(strip=True)
            parent = div.find_parent("div", class_="tmlogs_icon")

            if unit_name:
                text = f"{amount} {unit_name}"
            else:
                text = amount

            if parent:
                parent.replace_with(text)
            else:
                div.replace_with(text)

    # Replace specific amounts
    replace_amount("workers_amount", "workers")
    replace_amount("coins_amount", "coins")
    replace_amount("power_amount", "power")
    replace_amount("spade_amount", "spade(s)")
    replace_amount("vp_amount", "VP")
    replace_amount("priests_amount", "Priests") # Added priests_amount
    replace_amount("cult_p_amount", "priest(s)")

    # Cult track amounts (no unit needed as context usually provides it)
    replace_amount("earth_amount", "")
    replace_amount("fire_amount", "")
    replace_amount("water_amount", "")
    replace_amount("air_amount", "")

    # Replace terrain icons
    terrain_map = {
        "trans_mountains": "mountains",
        "trans_forest": "forest",
        "trans_lakes": "lakes",
        "trans_swamp": "swamp",
        "trans_desert": "desert",
        "trans_plains": "plains",
        "trans_wasteland": "wasteland",
    }

    for cls, name in terrain_map.items():
        for div in soup.find_all("div", class_=cls):
            parent = div.find_parent("div", class_="tmlogs_icon")
            if parent:
                parent.replace_with(name)
            else:
                div.replace_with(name)

    # General cleanup of tmlogs_icon if any remain (using title as fallback)
    for div in soup.find_all("div", class_="tmlogs_icon"):
        title = div.get("title")
        if title:
            div.replace_with(title)

    logs = soup.find_all("div", class_="gamelogreview")
    if not logs:
        return soup.get_text(separator=" ", strip=True)

    cleaned_logs = []
    for log in logs:
        text = log.get_text(separator=" ", strip=True)
        text = re.sub(r'\s+', ' ', text)

        # Fix conversion parenthesis: "... collects: ... ) ... ..." -> "... collects: ... ... ... )"
        # Pattern: look for "collects: ... )" followed by resources
        if "Conversions" in text and "collects:" in text:
            # Simple heuristic: if ')' is followed by numbers and words, move it to the end
            # Example: ... collects: 1 Priests ) 0 workers 0 coins
            # We want: ... collects: 1 Priests 0 workers 0 coins )

            # Find the part after "collects:"
            parts = text.split("collects:")
            if len(parts) > 1:
                suffix = parts[1]
                if ")" in suffix:
                    pre_paren, post_paren = suffix.split(")", 1)
                    # Check if post_paren contains resources (digits and words)
                    if re.search(r'\d+', post_paren):
                        # Move parenthesis to the end
                        new_suffix = pre_paren + post_paren + ")"
                        text = parts[0] + "collects:" + new_suffix

        cleaned_logs.append(text)

    return "\n".join(cleaned_logs)

def get_default_user_data_dir():
    """Get the default Chrome user data directory for persistent login."""
    home = os.path.expanduser("~")
    data_dir = os.path.join(home, ".tm_server", "chrome_profile")
    
    # Create directory if it doesn't exist
    if not os.path.exists(data_dir):
        os.makedirs(data_dir)
        print(f"Created Chrome profile directory: {data_dir}")
    
    return data_dir

def main():
    parser = argparse.ArgumentParser(description='Fetch BGA game logs.')
    parser.add_argument('table_id', help='The BGA table ID (e.g., 713277654)')
    parser.add_argument('--output', '-o', default='bga_log.txt', help='Output file path')
    parser.add_argument('--user-data-dir', help='Path to Chrome user data dir for persistent login (default: ~/.tm_server/chrome_profile)')
    parser.add_argument('--no-profile', action='store_true', help='Do not use persistent Chrome profile')
    args = parser.parse_args()

    url = f"https://boardgamearena.com/gamereview?table={args.table_id}"
    print(f"Fetching logs for table {args.table_id} from {url}...")

    chrome_options = Options()
    
    # Determine user data directory
    if args.no_profile:
        print("Using temporary Chrome profile (no persistence).")
        user_data_dir = None
    elif args.user_data_dir:
        user_data_dir = args.user_data_dir
        print(f"Using specified Chrome profile: {user_data_dir}")
    else:
        user_data_dir = get_default_user_data_dir()
        print(f"Using persistent Chrome profile: {user_data_dir}")
    
    if user_data_dir:
        chrome_options.add_argument(f"user-data-dir={user_data_dir}")

    # Anti-detection options
    chrome_options.add_argument("--disable-blink-features=AutomationControlled")
    chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
    chrome_options.add_experimental_option("useAutomationExtension", False)

    driver = webdriver.Chrome(options=chrome_options)

    # Execute CDP commands to further hide automation
    driver.execute_cdp_cmd("Page.addScriptToEvaluateOnNewDocument", {
        "source": """
            Object.defineProperty(navigator, 'webdriver', {
                get: () => undefined
            })
        """
    })

    try:
        driver.get(url)

        # Wait for user to log in if needed
        print("Waiting for page to load... (If redirected to login, please log in)")

        # Wait for the log container to be present.
        WebDriverWait(driver, 300).until(
            EC.presence_of_element_located((By.ID, "gamelogs"))
        )

        print("Game log container found. Extracting HTML...")

        # Allow some time for dynamic content to load
        time.sleep(5)

        # Extract HTML from the log container
        log_element = driver.find_element(By.ID, "gamelogs")
        log_html = log_element.get_attribute('innerHTML')

        print("Parsing HTML...")
        parsed_text = parse_log_html(log_html)

        # Save to file
        with open(args.output, 'w', encoding='utf-8') as f:
            f.write(parsed_text)

        print(f"Successfully saved {len(parsed_text)} characters to {args.output}")

    except Exception as e:
        print(f"Error: {e}")
    finally:
        driver.quit()

if __name__ == "__main__":
    main()
