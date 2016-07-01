
extract () {
  if [ -f $1 ] ; then
    case $1 in
      *.tar.bz2) tar xvjf $1   ;;
      *.tar.gz)  tar xvzf $1   ;;
      *.tar.xz)  tar xvfJ $1   ;;
      *.bz2)     bunzip2 $1    ;;
      *.rar)     unrar x $1    ;;
      *.gz)      gunzip $1     ;;
      *.tar)     tar xvf $1    ;;
      *.tbz2)    tar xvjf $1   ;;
      *.tgz)     tar xvzf $1   ;;
      *.zip)     unzip $1      ;;
      *.Z)       uncompress $1 ;;
      *.7z)      7z x $1       ;;
      *)         echo "'$1' cannot be extracted via >extract<" ;;
    esac
  else
    echo "'$1' is not a valid file"
  fi
}

myip() {
  curl -s http://www.formyip.com/ |grep The | awk {'print $5'}
}

cuttail() {  # удалить последние n строк в файле, по-умолчанию 10
    nlines=${2:-10}
    sed -n -e :a -e "1,${nlines}!{P;N;D;};N;ba" $1
}

lowercase() {  # перевести имя файла в нижний регистр
    for file ; do
        filename=${file##*/}
        case "$filename" in
        */*) dirname==${file%/*} ;;
        *) dirname=.;;
        esac
        nf=$(echo $filename | tr A-Z a-z)
        newname="${dirname}/${nf}"
        if [ "$nf" != "$filename" ]; then
            mv "$file" "$newname"
            echo "lowercase: $file --> $newname"
        else
            echo "lowercase: имя файла $file не было изменено."
        fi
    done
}

swap() {        # меняет 2 файла местами
    local TMPFILE=tmp.$$
    mv "$1" $TMPFILE
    mv "$2" "$1"
    mv $TMPFILE "$2"
}

function repeat() {       # повторить команду n раз
    local i max
    max=$1; shift;
    for ((i=1; i <= max ; i++)); do  # --> C-подобный синтаксис
        eval "$@";
    done
}

function ii() {   # Дополнительные сведения о системе
    echo -e "\nВы находитесь на $fg_bold[green]$HOST$reset_color"
    echo -e "\n$fg_bold[red]Дополнительная информация:$reset_color " ; uname -a
    echo -e "\n$fg_bold[red]В системе работают пользователи:$reset_color " ; w -h
    echo -e "\n$fg_bold[red]Дата:$reset_color " ; date
    echo -e "\n$fg_bold[red]Время, прошедшее с момента последней перезагрузки:$reset_color " ; uptime
#    echo -e "\n$fg_bold[red]Память:$reset_color " ; free
    echo
}
