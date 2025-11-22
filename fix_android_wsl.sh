#!/bin/bash

# 1. Clean up .bashrc
# Remove lines containing ANDROID_SDK_ROOT or ANDROID_HOME to start fresh
sed -i '/ANDROID_SDK_ROOT/d' ~/.bashrc
sed -i '/ANDROID_HOME/d' ~/.bashrc
sed -i '/cmdline-tools/d' ~/.bashrc
sed -i '/platform-tools/d' ~/.bashrc

# 2. Add correct environment variables
echo 'export ANDROID_HOME=$HOME/Android' >> ~/.bashrc
echo 'export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >> ~/.bashrc

# 3. Create directory
mkdir -p ~/Android/cmdline-tools
cd ~/Android/cmdline-tools

# 4. Download Linux Command Line Tools (if not exists)
if [ ! -d "latest" ]; then
    echo "Downloading Android Command Line Tools..."
    wget -q https://dl.google.com/android/repository/commandlinetools-linux-10406996_latest.zip -O tools.zip
    unzip -q tools.zip
    mv cmdline-tools latest
    rm tools.zip
fi

# 5. Install SDK components
# We need to source the new env vars for this session
export ANDROID_HOME=$HOME/Android
export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin
export PATH=$PATH:$ANDROID_HOME/platform-tools

echo "Installing Build Tools..."
yes | sdkmanager --licenses > /dev/null
sdkmanager "platform-tools" "platforms;android-34" "build-tools;34.0.0"

echo "Done! Please restart your terminal."
