# README #

### What is this repository for? ###

* commandline text image file generator

[![Build Status](https://drone.io/bitbucket.org/shu/mojimg/status.png)](https://drone.io/bitbucket.org/shu/mojimg/latest)

### How do I get set up? ###

* Download and go build
* Put a ttf file in "font" folder
	* The name of ttf file must end with "mr.ttf" (this is a restriction of draw2d)

### Binaries ###

[drone.io](https://drone.io/bitbucket.org/shu/mojimg/files)

### Usage ###

	#mojimg -h

	#mojimg -output test.jpg test1 test2
		=> render lines ("test1" and "test2") and output test.jpg
	#mojimg test1 test2  |  mojimg -pos middlecenter -fg #f00 -output test.jpg CENTER
		=> pipelined rendering
	#mojimg test.jpg test1 test2  |  wpchanger
		=> (Windows) change the wallpaper with [wpchanger](https://bitbucket.org/shu/wpchanger)

### Dependency ###

* github.com/llgcode/draw2d
	* rendering text
* github.com/andrew-d/go-termutil
	* isatty
