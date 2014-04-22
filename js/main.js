(function($){
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

  function parsePullRequest(username, headCommit, pullRequest) {
    saturatePullRequest(pullRequest).done(function(pullRequest) {
      pullRequest.iAmOwner = pullRequest.user.login == username;
      pullRequest.approvals = approvingComments(pullRequest.comments);
      pullRequest.numApprovals = Object.keys(pullRequest.approvals).length;
      pullRequest.iHaveApproved = !!pullRequest.approvals[username];
      pullRequest.isRebased = ancestryContains(pullRequest.commits, headCommit);
      pullRequest.state = getState(pullRequest.statuses);

      var html = buildRow(pullRequest);

      if (pullRequest.numApprovals >= 2) {
        $('#approved-prs tbody').prepend(html);
      } else {
        $('#approved-prs tbody').append(html);
      }
    });
  }

  function saturatePullRequest(pullRequest) {
    return $.when(
      $.ajax(pullRequest.comments_url),
      $.ajax(pullRequest.commits_url),
      $.ajax(pullRequest.statuses_url)
    ).then(function(comments, commits, statuses) {
      return $.extend(pullRequest, {comments: comments[0], commits: commits[0], statuses: statuses[0]});
    });
  };

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
    var result = {};
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

  function buildRow(pullRequest) {
    var row = '<td><a href="' + pullRequest.html_url + '">' + pullRequest.number + '</a></td><td title="' + approvalTitle(pullRequest) + '">' +
      pullRequest.numApprovals + '</td><td>' + (pullRequest.isRebased ? 'Y' : 'N') + '</td><td>' +
      (pullRequest.state == 'success' ? 'Y' : pullRequest.state == 'none' ? '?' : 'N') + '</td><td>' +
      (!pullRequest.iHaveApproved && !pullRequest.iAmOwner ? 'Y' : 'N') + '</td><td>' +
      (pullRequest.iAmOwner ? 'Y' : 'N') + '</td>';

    return '<tr class="' + rowClass(pullRequest) + '" data-link="' + pullRequest.html_url + '">' + row + + '</tr>';
  }

  function approvalTitle(pullRequest) {
    var title = '';
    for (var commentor in pullRequest.approvals) {
      for (var i in pullRequest.approvals[commentor]) {
        title += commentor + ': ' + pullRequest.approvals[commentor][i] + '\n';
      }
    }

    return title;
  }

  function rowClass(pullRequest) {
    if (pullRequest.numApprovals >= 2 && pullRequest.isRebased) {
      return 'success';
    }
    
    if (!pullRequest.iHaveApproved && !pullRequest.iAmOwner) {
      return 'info';
    }
    
    if (pullRequest.iAmOwner && !pullRequest.isRebased) {
      return 'warning';
    }

    if (pullRequest.state == 'failure') {
      return 'danger';
    }

    return '';
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
