import subprocess
import smtplib
from email.mime.text import MIMEText
import sys
from datetime import datetime

SMTP_USER = "helmetarmored@gmail.com"
SMTP_PASSWORD = "rfom llmd ovhk tkto"
SMTP_HOST = "smtp.gmail.com"
SMTP_PORT = 587

SCRIPTS = ["1_ground_truth_gemma.py", "2_prod_metadata_qwen.py", "3_4_vectorize.py", "5_optimize_weights.py", "6_evaluation.py"]

def send_email(subject, body):
    msg = MIMEText(body)
    msg['Subject'] = f"[{datetime.now().strftime('%H:%M')}] {subject}"
    msg['From'] = SMTP_USER
    msg['To'] = SMTP_USER
    try:
        with smtplib.SMTP(SMTP_HOST, SMTP_PORT) as server:
            server.starttls()
            server.login(SMTP_USER, SMTP_PASSWORD)
            server.send_message(msg)
    except Exception:
        pass

def main():
    for script in SCRIPTS:
        print(f"\n{'='*50}\nStarting {script}...\n{'='*50}")
        process = subprocess.Popen([sys.executable, script], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
        log_history = []
        for line in process.stdout:  # type: ignore
            sys.stdout.write(line)
            log_history.append(line)
        process.wait()
        
        if process.returncode != 0:
            print(f"\n[CRITICAL ERROR] {script} failed.")
            send_email(f"PIPELINE FAILURE: {script}", f"Crashed at: {script}\n\nLast Logs:\n" + "".join(log_history[-100:]))
            sys.exit(1)
        else:
            print(f"\n[SUCCESS] Finished {script}.")
            send_email(f"PIPELINE SUCCESS: {script}", f"Completed successfully.\n\nSummary:\n" + "".join(log_history[-10:]))

if __name__ == "__main__":
    main()

