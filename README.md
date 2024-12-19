im just using this so i dont have to rebuild this shit everytime i have some sideproject idea that i will never complete and leave it hanging

- uses tailwindcss
- uses go templating + htmx for client interaction with server
- all built into a single container

dependencies
1) tailwindcss, https://tailwindcss.com/docs/installation , follow standalone executable
   - note: if u are adding new folder for html files, please update tailwind.config.js to apply those changes
2) entr for hot reload, https://jvns.ca/blog/2020/06/28/entr/
3) htmx for simple client server interactivity 