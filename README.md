commandline text image file generator

[![Go Report Card](https://goreportcard.com/badge/github.com/shu-go/mojimg)](https://goreportcard.com/report/github.com/shu-go/mojimg)
![MIT License](https://img.shields.io/badge/License-MIT-blue)


# Usage

## Preparation

Place a ttf font file

## Rendering text and emoji

```
mojimg --font ./myfont.ttf --output output.png text1 text2 ::smile::
```

## Pipelining

```
cat(or type) output.png | mojimg --font ./myfont.ttf --output output.png --pos cm center-middle-text
```

## Help

```
mojimg help
```

<!-- vim: set et ft=markdown sts=4 sw=4 ts=4 tw=0 : -->
