@echo off
rem expecting path:
rem ./font/
rem    ipagmr.ttf
rem mojimg
rem sample.bat (this file)
@echo on

mojimg -bg 0ff -pos lt left top | mojimg -pos ct center top | mojimg -pos rt right top | mojimg -pos lm left middle | mojimg -fg f00 -pos cm center middle | mojimg -pos rm right middle | mojimg -pos lb left bottom | mojimg -pos cb center bottom | mojimg -pos rb right bottom > sample.png
type sample.png | mojimg -type=jpg -pos rb " "  > sample.jpg

mojimg -output sample_autodetection.jpg JPEG
mojimg -output sample_autodetection.png PNG
