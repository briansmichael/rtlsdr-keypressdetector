# rtlsdr-keypressdetector
SDR Keypress Detector

i# 1. Install rtl-sdr tools
sudo apt update && sudo apt install -y rtl-sdr

# 2. Allow non-root access to the SDR dongle
sudo cp /usr/lib/udev/rules.d/rtl-sdr.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules && sudo udevadm trigger

# 3. Install Go (if not already present)
wget https://go.dev/dl/go1.22.0.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 4. Build
cd rtlsdr-keypressdetector
go build -o keypressdetector .

# 5. Calibrate squelch threshold (Ctrl-C to exit)
./keypressdetector -calibrate

# 6. Run (GPIO8 HIGH for 10 min after 5 presses on 122.725 MHz)
./keypressdetector -squelch 250 -verbose

# 7. (Optional) Install as a systemd service
sudo cp keypressdetector.service /etc/systemd/system/
sudo systemctl enable --now keypressdetector
