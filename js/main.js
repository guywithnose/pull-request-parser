(function(window, $) {
  var apiUrl = 'https://api.github.com';
  var MIN_APPROVALS = 2;

  var LS = function(namespace) {
    this.keyOf = function(name) {
      return name + ':' + namespace;
    }
  }

  LS.prototype.getAccessToken = function() {
    return window.localStorage[this.keyOf('github_access_token')];
  }

  LS.prototype.setAccessToken = function(accessToken) {
    return window.localStorage[this.keyOf('github_access_token')] = accessToken;
  }

  LS.prototype.getRepoPaths = function() {
    var repos = window.localStorage[this.keyOf('repos')];

    return repos ? JSON.parse(repos) : [];
  }

  LS.prototype.addRepoPath = function(repoPath) {
    var repoPaths = this.getRepoPaths();
    if (repoPaths.indexOf(repoPath) === -1) {
      repoPaths.push(repoPath);
      localStorage[this.keyOf('repos')] = JSON.stringify(repoPaths);
    }
  }

  function updateSelectBoxes(repoPaths) {
    $('#repoPathSelect').html('<option></option>');
    if (repoPaths) {
      for (var i in repoPaths) {
          $('#repoPathSelect').append('<option>' + repoPaths[i] + '</option>');
      }

      $('#repoPathSelect').show();
    }
  }

  function parsePullRequests(repoPath) {
    $.when(
        $.ajax(apiUrl + '/user'),
        $.ajax(apiUrl + '/repos/' + repoPath + '/commits/master'),
        $.ajax(apiUrl + '/repos/' + repoPath + '/pulls')
    ).done(parseAllPullRequests);
  }

  function parseRepos(repoPaths) {
    if (repoPaths.indexOf(repoPath) == -1) {
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

  function refreshPr(repoPath, prNum) {
    $.when(
        $.ajax(apiUrl + '/user'),
        $.ajax(apiUrl + '/repos/' + repoPath + '/commits/master'),
        $.ajax(apiUrl + '/repos/' + repoPath + '/pulls/' + prNum)
    ).done(function(userXhr, masterXhr, pullRequestDataXhr) {
      parsePullRequest(userXhr[0].login, masterXhr[0].sha, pullRequestDataXhr[0]);
    });
  }

  function parsePullRequest(username, headCommit, pullRequest) {
    saturatePullRequest(pullRequest).then(function(pullRequest) {
      pullRequest.iAmOwner = pullRequest.user.login == username;
      pullRequest.approvals = approvingComments(pullRequest.comments);
      pullRequest.numApprovals = Object.keys(pullRequest.approvals).length;
      pullRequest.approved = pullRequest.numApprovals >= MIN_APPROVALS;
      pullRequest.iHaveApproved = !!pullRequest.approvals[username];
      pullRequest.isRebased = ancestryContains(pullRequest.commits, headCommit);
      pullRequest.rebasedText = pullRequest.isRebased ? 'Y' : 'N';
      var state = getState(pullRequest.statuses);
      pullRequest.state = state == 'success' ? 'Y' : state == 'none' || state == 'pending' ? '?' : 'N';
      pullRequest.needsMyApproval = !pullRequest.iHaveApproved && !pullRequest.iAmOwner ? 'Y' : 'N';

      buildHtml(pullRequest);
    }, function(pullRequest) {
      pullRequest.numApprovals = '?';
      pullRequest.iHaveApproved = false;
      pullRequest.isRebased = false;
      pullRequest.rebasedText = '?';
      pullRequest.state = '?';
      pullRequest.needsMyApproval = '?';
      buildHtml(pullRequest);
    });
  }

  function buildHtml(pullRequest) {
      var html = buildRow(pullRequest);

      if (pullRequest.approved) {
        $('#approved-prs tbody').prepend(html);
      } else {
        $('#approved-prs tbody').append(html);
      }
  }

  function saturatePullRequest(pullRequest) {
    // This is a crude way of detecting that we are on an old version of the API where the urls are broken and need to be built manually.
    if (pullRequest.statuses_url) {
      return $.when(
        $.ajax(pullRequest.comments_url),
        $.ajax(pullRequest.commits_url),
        $.ajax(pullRequest.statuses_url)
      ).then(function(comments, commits, statuses) {
        return $.extend(pullRequest, {comments: comments[0], commits: commits[0], statuses: statuses[0]});
      });
    }

    return $.when(
      $.ajax(pullRequest.comments_url),
      $.ajax(pullRequest.url + '/commits'),
      $.ajax(pullRequest.base.repo.statuses_url.replace('{sha}', pullRequest.head.sha))
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
   * Returns the users that have a comment containing :+1: or LGTM.
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
    return comment.body.search(':\\+1:') != -1 ||
      comment.body.search(':thumbsup:') != -1 ||
      comment.body.search('LGTM') != -1;
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
      '<td>' + pullRequest.rebasedText + '</td>' +
      '<td>' + pullRequest.state + '</td>' +
      '<td>' + pullRequest.needsMyApproval + '</td>' +
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
    if (pullRequest.approved && pullRequest.isRebased) {
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

  var PullRequestParser = function(options) {
    options = options || {};

    apiUrl = options.apiUrl || apiUrl;
    MIN_APPROVALS = options.minApprovals || MIN_APPROVALS;

    $.ajaxSetup({
      dataType: "json",
      cache: false
    });

    var ls = new LS(apiUrl);

    if (ls.getAccessToken()) {
      $.ajaxSetup({
        headers: {Authorization: 'token ' + ls.getAccessToken()}
      });
      $('#pickRepo').show();
    } else {
      $('#getAccessToken').show();
    }

    updateSelectBoxes(ls.getRepoPaths());

    $('#saveAccessToken').click(function() {
      ls.setAccessToken($('#accessToken').val());
      $.ajaxSetup({
        headers: {Authorization: 'token ' + ls.getAccessToken()}
      });
      $('#getAccessToken').hide();
      $('#pickRepo').show();
    });

    $('#parsePullRequests').click(function() {
      var repoPaths = $('#repoPath').val().split('\n');

      $.each(repoPaths, function(index, repoPath) {
        ls.addRepoPath(repoPath);
      });
      updateSelectBoxes(ls.getRepoPaths());

      $('#approved-prs tbody').html('');
      parseRepos(repoPaths);
    });

    $('#checkAllRepos').click(function() {
      $('#approved-prs tbody').html('');
      parseRepos(ls.getRepoPaths());
    });

    $('#repoPathSelect').change(function(){
      $('#repoPath').val($(this).val());
    });

    $('#approved-prs').on('click', '.refresh', function() {
      var row = $(this).parents('tr');
      var repoPath = row.data('repoPath');
      var prNum = row.data('prNum');
      $(this).parent().parent().remove();
      refreshPr(repoPath, prNum);
    });
  };

  window.PullRequestParser = PullRequestParser;
}(window, jQuery))
