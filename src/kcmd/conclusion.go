package kcmd

func GrubConclusion(grubFile, old, new string) []string {
	red := "\033[31m"
	green := "\033[32m"
	reset := "\033[0m"

	s := []string{
		"Detected bootloader: GRUB\n",
		"Default kernel command line:\n",
		red + "-  " + old + reset + "\n",
		"New kernel command line:\n",
		green + "+  " + new + reset + "\n",
		"Updated default grub file: " + grubFile + "\n",
		"\n",
		"Please run:\n",
		"\n",
		"\tsudo update-grub\n",
		"\n",
		"to apply the changes to your bootloader.\n",
		"\n",
	}
	return s
}

func RpiConclusion(cmdline string) []string {
	s := []string{
		"Detected bootloader: Raspberry Pi\n",
		"\n",
		"Please, append the following to /boot/firmware/cmdline.txt:\n",
		"In case of old style boot partition,\n",
		"append to /boot/cmdline.txt\n",
		cmdline,
		"\n",
	}
	return s
}

func UbuntuCoreConclusion() []string {
	s := []string{
		"Detected bootloader: Ubuntu Core managed\n",
		"\n",
		"Sucessfully applied the changes.\n",
		"Please reboot your system to apply the changes.\n",
		"\n",
	}
	return s
}
