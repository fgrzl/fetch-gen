// Auto-generated by fetch-gen

import client from '@fgrzl/fetch';


export const getUser = (): Promise<any> =>
  client.get(`/users`);

export const createUser = (body: User): Promise<any> =>
  client.post(`/users`, body);

export const updateUser = (id: string, body: User): Promise<any> =>
  client.put(`/users/${id}`, body);

export const deleteUser = (id: string): Promise<any> =>
  client.delete(`/users/${id}`);



export interface User {
  id: string;
  metadata: Record<string, string>;
  name: string;
  profile: { bio: string; age: number };
  status: "active" | "inactive" | "banned";
  tags: Array<string>;
}

