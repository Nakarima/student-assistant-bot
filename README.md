# student-assistant-bot
Student assistant telegram bot written in Go

## What is it?

It's simple chatbot that helps students in learning and organizing. You use it in Telegram, so you don't need to install additional apps or remember new passwords. Bot can help you by studying from flashcards on the go, remembering your schedule, reminding about exams and finding definitions on Wikipedia. It can be used in groups so you can share knowledge with your friends.

## Available functions

* **/dodajfiszke** - starts a dialog with bot to add a new flashcard. He will ask for topic, term and definition. You can have same terms under different subjects.
* **/fiszka _term_** - bot will give you definition (or definitions) for given term. 
* **/edytujfiszke** - starts a dialog with bot to edit an existing flashcard. He will ask for topic, term and if flashcard exists it will ask for new definition.
* **/usunfiszke** - starts a dialog with bot to delete an existing flashcard. He will ask for topic and term. 
* **/version** - bot will print his current version.
