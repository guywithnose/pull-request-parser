(function($){
  var username;

  function updateSelectBoxes() {
    $('#owners').html('<option></option>');
    $('#repos').html('<option></option>');
    if (localStorage.owners) {
      var owners = JSON.parse(localStorage.owners);
      for (var i in owners) {
          $('#owners').append('<option>' + owners[i] + '</option>');
      }

      $('#owners').show();
    }

    if (localStorage.repos) {
      var repos = JSON.parse(localStorage.repos);
      for (var i in repos) {
          $('#repos').append('<option>' + repos[i] + '</option>');
      }

      $('#repos').show();
    }
  }

  function addFavorites(newOwner, newRepo) {
      var owners;
      if (localStorage.owners) {
        owners = JSON.parse(localStorage.owners);
      } else {
        owners = [];
      }

      if (owners.indexOf(newOwner) == -1) {
        owners.push(newOwner);
      }

      localStorage.owners = JSON.stringify(owners);

      var repos;
      if (localStorage.repos) {
        repos = JSON.parse(localStorage.repos);
      } else {
        repos = [];
      }

      if (repos.indexOf(newRepo) == -1) {
        repos.push(newRepo);
      }

      localStorage.repos = JSON.stringify(repos);

      updateSelectBoxes();
  }

  function parsePullRequests(owner, repo) {
    $('#approved-prs tbody').html('');
    $.when(
        $.ajax('https://api.github.com/user'),
        $.ajax('https://api.github.com/repos/' + owner + '/' + repo + '/commits/master'),
        $.ajax('https://api.github.com/repos/' + owner + '/' + repo + '/pulls')
    ).done(function(userXhr, masterXhr, pullRequestDataXhr) {
      var username = userXhr[0].login;
      var headCommit = masterXhr[0].sha;
      for (var i in pullRequestDataXhr[0]) {
        parsePullRequest(username, headCommit, pullRequestDataXhr[0][i]);
      }
    });
  }

  function parsePullRequest(username, headCommit, pullRequestData) {
    $.when(
        $.ajax(pullRequestData.comments_url),
        $.ajax(pullRequestData.commits_url),
        $.ajax(pullRequestData.statuses_url)
    ).done(function(commentsXhr, commitsXhr, statusesXhr) {
      var comments = commentsXhr[0];
      var commits = commitsXhr[0];
      var statuses = statusesXhr[0];

      var iAmOwner = pullRequestData.user.login == username;
      var approvals = approvingComments(comments);
      var iHaveApproved = $.inArray(username, approvals) !== -1;
      var isRebased = ancestryContains(commits, headCommit);

      var html = buildRow(approvals, pullRequestData, iHaveApproved, isRebased, getState(statuses), iAmOwner);

      if (approvals.length >= 2) {
        $('#approved-prs tbody').prepend(html);
      } else {
        $('#approved-prs tbody').append(html);
      }
    });
  }

  function getState(statuses) {
    if (statuses.length == 0) {
      return 'none';
    }

    return statuses[0].state;
  }

  /*
   * Returns the users that have a comment containing :+1:.
   */
  function approvingComments(comments) {
    var result = [];
    for (var i in comments) {
      if (comments[i].body.search(':\\+1:') != -1 && $.inArray(comments[i].user.login, result) === -1) {
        if (!result[comments[i].user.login]) {
          result[comments[i].user.login] = [];
        }

        result[comments[i].user.login].push(comments[i].body);
      }
    }

    return result;
  }

  /*
   * Searches through the commits and checks to see if any of them contain the requested commit hash
   */
  function ancestryContains(commits, commitHash) {
    for (var i in commits) {
      for (var j in commits[i].parents) {
        var parent = commits[i].parents[j];
        if (parent.sha == commitHash) {
          return true;
        }
      }
    }

    return false;
  }

  function buildRow(approvals, pullRequestData, iHaveApproved, isRebased, state, iAmOwner, table) {
    var rowClass = '';
    var numApprovals = Object.keys(approvals).length;
    if (numApprovals >= 2 && isRebased) {
      rowClass = 'success';
    }
    
    if (!iHaveApproved && !iAmOwner) {
      rowClass = 'info';
    }
    
    if (iAmOwner && !isRebased) {
      rowClass = 'warning';
    }

    if (state == 'failure') {
      rowClass = 'danger';
    }

    var approvalTitle = '';
    for (var commentor in approvals) {
      for (var i in approvals[commentor]) {
        approvalTitle += commentor + ': ' + approvals[commentor][i] + '\n';
      }
    }

    var row = '<td><a href="' + pullRequestData.html_url + '">' + pullRequestData.number + '</a></td><td title="' + approvalTitle + '">' + numApprovals +
      '</td><td>' + (isRebased ? 'Y' : 'N') + '</td><td>' + (state == 'success' ? 'Y' : state == 'none' ? '?' : 'N') + '</td><td>' +  (!iHaveApproved && !iAmOwner ? 'Y' : 'N') + '</td><td>' + (iAmOwner ? 'Y' : 'N') + '</td>';

    return '<tr class="' + rowClass + '" data-link="' + pullRequestData.html_url + '">' + row + + '</tr>';
  }

  function init() {
    $.ajaxSetup({
      dataType: "json"
    });

    if (localStorage.github_access_token) {
      $.ajaxSetup({
        headers: {Authorization: 'token ' + localStorage.github_access_token}
      });
      $('#pickRepo').show();
    } else {
      $('#getAccessToken').show();
    }

    updateSelectBoxes();

    $('#saveAccessToken').click(function() {
      localStorage.github_access_token = $('#accessToken').val();
      $.ajaxSetup({
        headers: {Authorization: 'token ' + localStorage.github_access_token}
      });
      $('#getAccessToken').hide();
      $('#pickRepo').show();
    });

    $('#parsePullRequests').click(function() {
      var owner = $('#owner').val();
      var repo = $('#repoName').val();

      addFavorites(owner, repo);

      parsePullRequests(owner, repo);
    });

    $('#owners').change(function(){
      $('#owner').val($(this).val());
    });

    $('#repos').change(function(){
      $('#repoName').val($(this).val());
    });
  }

  $(init);
}(jQuery))
