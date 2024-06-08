# go-discord-pingpong
このプログラムはDiscordBotとしてpingpongします。

ping,pongに加え、ChatGPT APIによる返答が含まれています。

![screenshot](https://github.com/go-numb/go-discord-pingpong/blob/images/sc.png)

## Envs
// 環境変数からの取得  
DISCORDBOTTOKEN string  
CHATGPTAPITOKEN string  
BOTID           string  



## Usage
```sh
$ git clone https://github.com/go-numb/go-discord-pingpong.git
$ cd this_dir
$ set DISCORDBOTTOKEN="" // or export
$ set CHATGPTTOKEN=""
$ set BOTID=""
$ go build or go run main.go
```


## Author

[@_numbP](https://twitter.com/_numbP)