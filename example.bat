@echo off
set /p fontpath=Font Path: 
set font=-font %fontpath%
@echo on

mojimg %font% -bg 0ff -pos lt left top | mojimg %font% -pos ct center top | mojimg %font% -pos rt right top | mojimg %font% -pos lm left middle | mojimg %font% -fg f00 -pos cm center ::smile:: middle | mojimg %font% -pos rm right middle | mojimg %font% -pos lb left bottom | mojimg %font% -pos cb center bottom | mojimg %font% -pos rb right bottom > sample.png
mojimg %font% -output sample_autodetection.png PNG
