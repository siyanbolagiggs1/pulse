"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  trustScore: number;
  suspended: boolean;
  createdAt: string;
}

const PAGE_SIZE = 20;

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [role, setRole] = useState("all");
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [acting, setActing] = useState<string | null>(null);

  useEffect(() => { setPage(1); }, [role]);

  useEffect(() => {
    const params: Record<string, string | number> = { page, limit: PAGE_SIZE };
    if (role !== "all") params.role = role;
    setLoading(true);
    adminApi.listUsers(params)
      .then((r) => {
        setUsers(r.data.data);
        setPages(r.data.meta?.pages ?? 1);
      })
      .catch(() => toast({ title: "Failed to load users", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [role, page]);

  const handleSuspend = async (id: string) => {
    setActing(id);
    try {
      await adminApi.suspendUser(id);
      toast({ title: "User suspended" });
      setUsers((prev) => prev.map((u) => u.id === id ? { ...u, suspended: true } : u));
    } catch {
      toast({ title: "Action failed", variant: "destructive" });
    } finally { setActing(null); }
  };

  const handleUnsuspend = async (id: string) => {
    setActing(id);
    try {
      await adminApi.unsuspendUser(id);
      toast({ title: "User unsuspended" });
      setUsers((prev) => prev.map((u) => u.id === id ? { ...u, suspended: false, trustScore: 50 } : u));
    } catch {
      toast({ title: "Action failed", variant: "destructive" });
    } finally { setActing(null); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Users</h2>
          <p className="text-muted-foreground">Manage platform users</p>
        </div>
        <Select value={role} onValueChange={setRole}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Roles</SelectItem>
            <SelectItem value="business">Business</SelectItem>
            <SelectItem value="promoter">Promoter</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">{[...Array(5)].map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
          ) : users.length === 0 ? (
            <p className="py-12 text-center text-muted-foreground">No users found</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Trust Score</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((u) => (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">{u.name}</TableCell>
                      <TableCell className="text-muted-foreground text-sm">{u.email}</TableCell>
                      <TableCell><Badge variant="secondary" className="capitalize">{u.role}</Badge></TableCell>
                      <TableCell>
                        <span className={u.trustScore < 30 ? "text-destructive" : u.trustScore < 60 ? "text-yellow-400" : "text-green-400"}>
                          {u.trustScore}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant={u.suspended ? "destructive" : "success"}>
                          {u.suspended ? "Suspended" : "Active"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">{format(new Date(u.createdAt), "MMM d, yyyy")}</TableCell>
                      <TableCell>
                        {u.role !== "admin" && (
                          u.suspended ? (
                            <Button size="sm" variant="outline" onClick={() => handleUnsuspend(u.id)} disabled={acting === u.id}>
                              Unsuspend
                            </Button>
                          ) : (
                            <Button size="sm" variant="destructive" onClick={() => handleSuspend(u.id)} disabled={acting === u.id}>
                              Suspend
                            </Button>
                          )
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="px-6 pb-4">
                <Pagination page={page} pages={pages} onChange={setPage} />
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
