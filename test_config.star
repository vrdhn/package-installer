print("OS:", get_os())
print("Arch:", get_arch())

def install_vlc(package_name):
    print("Installing VLC:", package_name)

def install_firefox(package_name):
    print("Installing Firefox:", package_name)

add_package("^vlc", install_vlc)
add_package("^firefox", install_firefox)
print("Packages registered successfully.")