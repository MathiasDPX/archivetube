/* Global UI interactions */
(function () {
    function closeDropdowns(except) {
        document.querySelectorAll('.dropdown.open').forEach(function (d) {
            if (except && d === except) return;
            d.classList.remove('open');
        });
    }

    function closeDetails(except) {
        document.querySelectorAll('details.nav-dropdown[open], details.action-menu[open]')
            .forEach(function (d) {
                if (except && d === except) return;
                d.removeAttribute('open');
            });
    }

    document.addEventListener('click', function (e) {
        var toggle = e.target.closest && e.target.closest('.dropdown-toggle');
        if (toggle) {
            var dd = toggle.closest('.dropdown');
            if (!dd) return;
            var wasOpen = dd.classList.contains('open');
            closeDropdowns(dd);
            if (!wasOpen) dd.classList.add('open');
        } else {
            closeDropdowns();
        }

        var details = e.target.closest && e.target.closest('details.nav-dropdown, details.action-menu');
        if (details) {
            closeDetails(details);
        } else {
            closeDetails();
        }
    });
})();
