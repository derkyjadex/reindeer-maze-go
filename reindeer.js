(function ($) {
    'use strict';

    var cells;
    var lastPlayers = [];
    var playerList = [];

    function renderMaze(maze) {
        var table = $('<table>');
        cells = [];
        for (var y = maze.height - 1; y >= 0; y--) {
            var row = $('<tr>');
            cells[y] = [];
            for (var x = 0; x < maze.width; x++) {
                cells[y][x] = $('<td>')
                    .appendTo(row);
            }

            row.appendTo(table);
        }

        cells[maze.presentY][maze.presentX].addClass('present');

        $.each(maze.walls, function (_, wall) {
            cells[wall[1]][wall[0]].addClass('wall');
        });

        table.appendTo('body');
    }

    function updatePlayers(players) {
        $.each(lastPlayers, function (_, player) {
            cells[player.y][player.x].removeClass('player');
        });
        playerList.empty();

        $.each(players, function (_, player) {
            cells[player.y][player.x].addClass('player');
            $('<li>')
                .text(player.name)
                .appendTo(playerList);
        });

        lastPlayers = players;
    }

    function getPlayers() {
        $.getJSON('players')
            .done(function (data) {
                updatePlayers(data);
                setTimeout(getPlayers, 200);
            });
    }

    $(function () {
        var infoDiv = $('<div>')
            .text('Loading...')
            .appendTo('body');

        $.getJSON('maze')
            .done(function (data) {
                infoDiv.text('');
                renderMaze(data);
                playerList = $('<ul>')
                    .appendTo('body');
                getPlayers();
            })
            .fail(function () {
                infoDiv.text('Error loading maze data');
            });
    });
})(jQuery);
