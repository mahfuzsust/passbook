package main

func ensureKDFParams() {
	if kdfTime == 0 {
		kdfTime = 3
	}
	if kdfMemoryKB == 0 {
		kdfMemoryKB = 64 * 1024
	}
	if kdfThreads == 0 {
		kdfThreads = 2
	}
}
