Output assets/gixt-combined-demo.gif
Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 560
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 45ms

Sleep 1s
Type `gixt add leolaurindo/hello-world.sh --as hello`
Enter
Sleep 3s

Type `gixt list`
Enter
Sleep 2s

Type `gixt run hello`
Enter
Sleep 1s
Type `y`
Enter
Sleep 2s

Type `gixt cat hello; echo`
Enter
Sleep 3s

Type `gixt add leolaurindo/go-style.md --as go-style`
Enter
Sleep 2s

Type `gixt add leolaurindo/docs-style-guide.md --as docs`
Enter
Sleep 2s

Type `gixt list`
Enter
Sleep 2s

Type `gixt cat go-style docs >> AGENTS.md`
Enter
Sleep 3s

Type `cat AGENTS.md`
Enter
Sleep 3s

Type `gixt add leolaurindo/capybara.txt --as capybara`
Enter
Sleep 2s

Type `gixt cat capybara`
Enter
Sleep 5s
