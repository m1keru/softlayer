# bash completion for softlayer
# Install: softlayer completion bash > /etc/bash_completion.d/softlayer
# or:      source <(softlayer completion bash)

_softlayer_cmd() {
    case "$1" in
        -list)     echo list ;;
        -lease)     echo lease ;;
        -liststale) echo stale ;;
        *)          echo "$1" ;;
    esac
}

_softlayer() {
    local cur prev cmd
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmd="$(_softlayer_cmd "${COMP_WORDS[1]}")"

    if [[ $COMP_CWORD -eq 1 ]]; then
        if [[ "$cur" == -* ]]; then
            COMPREPLY=( $(compgen -W "-list -lease -liststale -ip -version -h --help" -- "$cur") )
        else
            COMPREPLY=( $(compgen -W "list set clear lease stale completion version help" -- "$cur") )
        fi
        return
    fi

    case "$prev" in
        -ip|-ptr|-note|-ttl|-search|-exclude-note|-exclude-cidr|--ip|--ptr|--note|--ttl|--search|--exclude-note|--exclude-cidr)
            return
            ;;
    esac

    case "$cmd" in
        list)
            COMPREPLY=( $(compgen -W "-public -private -all -one -json -exclude-note -exclude-cidr -no-default-excludes -h --help" -- "$cur") )
            ;;
        set)
            COMPREPLY=( $(compgen -W "-ip -ptr -note -ttl -force -h --help" -- "$cur") )
            ;;
        clear)
            COMPREPLY=( $(compgen -W "-ip -force -h --help" -- "$cur") )
            ;;
        lease)
            COMPREPLY=( $(compgen -W "-ptr -note -ttl -search -force -exclude-note -exclude-cidr -no-default-excludes -h --help" -- "$cur") )
            ;;
        stale)
            COMPREPLY=( $(compgen -W "-json -exclude-note -exclude-cidr -no-default-excludes -h --help" -- "$cur") )
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
            ;;
        version|help)
            COMPREPLY=()
            ;;
    esac
}

complete -F _softlayer softlayer
