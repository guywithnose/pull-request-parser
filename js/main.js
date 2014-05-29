(function($){
  function updateSelectBoxes() {
    $('#repoPathSelect').html('<option></option>');
    if (localStorage.repoPaths) {
      var repoPaths = JSON.parse(localStorage.repoPaths);
      for (var i in repoPaths) {
          $('#repoPathSelect').append('<option>' + repoPaths[i] + '</option>');
      }

      $('#repoPathSelect').show();
    }

    if (localStorage.repos) {
      var repos = JSON.parse(localStorage.repos);
      for (var i in repos) {
          $('#repos').append('<option>' + repos[i] + '</option>');
      }

      $('#repos').show();
    }
  }

  function addFavorites(repoPath) {
      var repoPaths;
      if (localStorage.repoPaths) {
        repoPaths = JSON.parse(localStorage.repoPaths);
      } else {
        repoPaths = [];
      }

      if (repoPaths.indexOf(repoPath) == -1) {
        repoPaths.push(repoPath);
      }

      localStorage.repoPaths = JSON.stringify(repoPaths);

      updateSelectBoxes();
  }

  function parsePullRequests(repoPath) {
    $.when(
        $.ajax('https://api.github.com/user'),
        $.ajax('https://api.github.com/repos/' + repoPath + '/commits/master'),
        $.ajax('https://api.github.com/repos/' + repoPath + '/pulls')
    ).done(parseAllPullRequests);
  }

  function parseAllRepos() {
    if (localStorage.repoPaths) {
      var repoPaths = JSON.parse(localStorage.repoPaths);
      $.each(repoPaths, function(index, repoPath) {
        parsePullRequests(repoPath);
      });
    }
  }

  function parseAllPullRequests(userXhr, masterXhr, pullRequestDataXhr) {
    var username = userXhr[0].login;
    var headCommit = masterXhr[0].sha;
    for (var i in pullRequestDataXhr[0]) {
      parsePullRequest(username, headCommit, pullRequestDataXhr[0][i]);
    }
  }

  function refreshPr() {
    var row = $(this).parents('tr');
    var repoPath = row.data('repoPath');
    var prNum = row.data('prNum');
    $(this).parent().parent().remove();
    $.when(
        $.ajax('https://api.github.com/user'),
        $.ajax('https://api.github.com/repos/' + repoPath + '/commits/master'),
        $.ajax('https://api.github.com/repos/' + repoPath + '/pulls/' + prNum)
    ).done(function(userXhr, masterXhr, pullRequestDataXhr) {
      parsePullRequest(userXhr[0].login, masterXhr[0].sha, pullRequestDataXhr[0]);
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
      if (isApproval(comments[i]) && $.inArray(comments[i].user.login, result) === -1) {
        if (!result[comments[i].user.login]) {
          result[comments[i].user.login] = [];
        }

        result[comments[i].user.login].push(comments[i].body);
      }
    }

    return result;
  }

  function isApproval(comment) {
    return comment.body.search(':\\+1:') != -1 || comment.body.search(':thumbsup:') != -1;
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
    var row = '<td>' + pullRequest.base.repo.full_name + '</td>' +
      '<td><a href="' + pullRequest.html_url + '" target="_blank">' + pullRequest.number + '</a></td>' +
      '<td>' + pullRequest.user.login + '</td>' +
      '<td>' + pullRequest.head.ref + '</td>' +
      '<td title="' + approvalTitle(pullRequest) + '">' + pullRequest.numApprovals + '</td>' +
      '<td>' + (pullRequest.isRebased ? 'Y' : 'N') + '</td>' +
      '<td>' + (pullRequest.state == 'success' ? 'Y' : pullRequest.state == 'none' || pullRequest.state == 'pending' ? '?' : 'N') + '</td>' +
      '<td>' + (!pullRequest.iHaveApproved && !pullRequest.iAmOwner ? 'Y' : 'N') + '</td>' +
      '<td><button class="refresh">Refresh</button></td>';

    return '<tr data-pr-num="' + pullRequest.number + '" data-repo-path="' + pullRequest.base.repo.full_name + '" class="' + rowClass(pullRequest) + '" data-link="' + pullRequest.html_url + '">' + row + + '</tr>';
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
    
    if (pullRequest.iAmOwner && !pullRequest.isRebased && pullRequest.numApprovals >= 2) {
      return 'warning';
    }

    if (pullRequest.state == 'failure') {
      return 'danger';
    }

    return '';
  }

  function init() {
    $.ajaxSetup({
      dataType: "json",
      cache: false
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
      var repoPath = $('#repoPath').val();

      addFavorites(repoPath);

      $('#approved-prs tbody').html('');
      parsePullRequests(repoPath);
    });

    $('#checkAllRepos').click(function() {
      $('#approved-prs tbody').html('');
      parseAllRepos(repoPath);
    });

    $('#repoPathSelect').change(function(){
      $('#repoPath').val($(this).val());
    });

    $('#approved-prs').on('click', '.refresh', refreshPr);
  }

  $(init);
}(jQuery))
