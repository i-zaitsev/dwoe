(function () {
	var stick = true;

	window.addEventListener("scroll", function () {
		stick = window.innerHeight + window.scrollY >= document.body.scrollHeight - 120;
	});

	function follow(el) {
		new MutationObserver(function () {
			if (stick) window.scrollTo(0, document.body.scrollHeight);
		}).observe(el, { childList: true });
	}

	document.addEventListener("htmx:load", function (e) {
		var root = e.target;
		if (root.matches && root.matches("[data-follow]")) follow(root);
		if (root.querySelectorAll) root.querySelectorAll("[data-follow]").forEach(follow);
	});
})();
