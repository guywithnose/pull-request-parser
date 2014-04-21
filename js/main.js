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
    $('#approved-prs').html('');
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
      var approvals = 0;
      var iHaveApproved = false;
      for (var i in comments) {
        if (comments[i].body.search(':\\+1:') != -1) {
          approvals++;
          if (comments[i].user.login == username) {
            iHaveApproved = true;
          }
        }
      }

      var isRebased = false;
      for (var i in commits) {
        for (var j in commits[i].parents) {
          var parent = commits[i].parents[j];
          if (parent.sha == headCommit) {
            isRebased = true;
            break;
          }
        }
      }

      var html = buildDiv(approvals, pullRequestData, iHaveApproved, isRebased, statuses[0].state);

      if (approvals >= 2) {
        $('#approved-prs').prepend(html);
      } else {
        $('#approved-prs').append(html);
      }
    });
  }

  function buildDiv(approvals, pullRequestData, iHaveApproved, isRebased, state) {
    var aTag = '<a href="' + pullRequestData.html_url + '">Pull Request ' + pullRequestData.number + ' has ' + approvals +
      ' approvals and it is' + (isRebased ? '' : ' not') + ' rebased and the build ' + (state == 'success' ? 'was successful' : 'failed') + '</a>';

    if (approvals >= 2) {
      $('#approved-prs').prepend('<div style="font-weight:bold;">' + aTag + '</div>');
    } else {
      $('#approved-prs').append('<div>' + aTag + (iHaveApproved ? '' : '<span style="font-weight:bold;">Needs your approval</span>') + '</div>');
    }
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
