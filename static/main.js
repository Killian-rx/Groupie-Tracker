function Clickimg(artistname) {
    window.location.href = '/search?search=' + artistname + '&searchType=name';
}

function toggleMenu() {
  const menu = document.getElementById("searchby");
  if (menu.style.display === "none") {
      menu.style.display = "block";
  } else {
      menu.style.display = "none";
  }
}




